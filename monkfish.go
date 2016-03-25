package monkfish

import (
	"flag"
	"fmt"
	hash "hash/adler32"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/pelletier/go-toml"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack"
	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
)

type MonkConf struct {
	username   string
	password   string
	tenantName string
	authUrl    string
	region     string

	domain          string
	internalDomain  string
	lanIPPrefix     string
	useOnlyHostname bool
}

func (c *MonkConf) Parse(path string) error {
	config, err := toml.LoadFile(path)
	if err != nil {
		return err
	}

	c.username = config.Get("default.os_username").(string)
	c.password = config.Get("default.os_password").(string)
	c.tenantName = config.Get("default.os_tenant_name").(string)
	c.authUrl = config.Get("default.os_auth_url").(string)
	if config.Has("default.os_region") {
		c.region = config.Get("default.os_region").(string)
	} else {
		c.region = "RegionOne"
	}

	c.domain = config.Get("default.domain").(string)
	c.internalDomain = config.Get("default.internal_domain").(string)
	if config.Has("default.lan_ip_prefix") {
		c.lanIPPrefix = config.Get("default.lan_ip_prefix").(string)
	} else {
		c.lanIPPrefix = ""
	}
	if config.Has("default.use_only_hostname") {
		c.useOnlyHostname = config.Get("default.use_only_hostname").(bool)
	} else {
		c.useOnlyHostname = false
	}

	return nil
}

func Run() error {
	var configPath string
	var commitsToFile bool
	var target string
	var verbose bool
	var showsVersion bool
	var randomDelay int

	flag.BoolVar(&commitsToFile, "w", false, "Write to file")
	flag.StringVar(&target, "t", "/etc/hosts", "Target file to write hosts")
	flag.BoolVar(&verbose, "V", false, "Verbose mode")
	flag.StringVar(&configPath, "c", "/etc/monkfish.ini", "Config path")
	flag.BoolVar(&showsVersion, "version", false, "Just show version and quit")
	flag.IntVar(&randomDelay, "random-delay", 0, "Random delay before to access OpenStack API, in second")
	flag.Parse()

	if showsVersion {
		showVersion()
	}

	loggerf := newLoggerf(verbose)

	conf := &MonkConf{}
	if err := conf.Parse(configPath); err != nil {
		return err
	}

	if randomDelay > 0 {
		hostname, _ := os.Hostname()
		ha := hash.Checksum([]byte(hostname))
		rand.Seed(time.Now().UnixNano() + int64(ha))
		delay := rand.Intn(randomDelay)
		loggerf("Sleeping in %d seconds...", delay)
		time.Sleep(time.Duration(delay) * time.Second)
	}

	auth, err := openstack.AuthenticatedClient(gophercloud.AuthOptions{
		IdentityEndpoint: conf.authUrl,
		Username:         conf.username,
		Password:         conf.password,
		TenantName:       conf.tenantName,
	})
	if err != nil {
		return err
	}
	cli, err := openstack.NewComputeV2(auth, gophercloud.EndpointOpts{Region: conf.region})
	if err != nil {
		return err
	}

	res, err := servers.List(cli, &servers.ListOpts{Status: "ACTIVE"}).AllPages()
	if err != nil {
		return err
	}
	svs, err := servers.ExtractServers(res)
	if err != nil {
		return err
	}

	var targetIo io.Writer
	if commitsToFile {
		targetIo, err = ioutil.TempFile("", "monkfish-work--")
		if err != nil {
			return err
		}
	} else {
		targetIo = os.Stdout
	}

	src, err := os.Open("/etc/hosts.base")
	if err == nil {
		data, _ := ioutil.ReadAll(src)
		targetIo.Write(data)
		targetIo.Write([]byte("\n"))
	}

	for _, i := range svs {
		if i.Name == "" {
			loggerf("skip: [%s]%s\n", i.ID, i.Name)
			continue
		}
		loggerf("name: %s\n", i.Name)
		var name string
		if conf.useOnlyHostname {
			name = strings.Split(i.Name, ".")[0]
		} else {
			name = i.Name
		}

		if wan := findWanIP(i.Addresses); wan != "" {
			fmt.Fprintf(
				targetIo,
				"%s\t\t%s.%s\n",
				wan,
				name,
				conf.domain,
			)
		}
		if lan := findLanIP(i.Addresses, conf.lanIPPrefix); lan != "" {
			fmt.Fprintf(
				targetIo,
				"%s\t\t%s.%s\n",
				lan,
				name,
				conf.internalDomain,
			)
		}
	}

	if commitsToFile {
		if f, ok := targetIo.(*os.File); ok {
			tmppath := f.Name()
			f.Close()
			os.Chmod(tmppath, 0644)

			loggerf("Rename %s to %s\n", tmppath, target)
			err = os.Rename(tmppath, target)
			if err != nil {
				return err
			}

			defer os.Remove(tmppath)
		}
	}
	loggerf("Complete!\n")

	return nil
}

func newLoggerf(verbose bool) func(string, ...interface{}) {
	var out io.Writer
	if verbose {
		out = os.Stderr
	} else {
		out = ioutil.Discard
	}
	return func(f string, v ...interface{}) {
		fmt.Fprintf(out, f, v...)
	}
}

// FIXME: make this smart
var privateIPPrefix = []string{
	"10.",
	"172.16.",
	"172.17.",
	"172.18.",
	"172.19.",
	"172.20.",
	"172.21.",
	"172.22.",
	"172.23.",
	"172.24.",
	"172.25.",
	"172.26.",
	"172.27.",
	"172.28.",
	"172.29.",
	"172.30.",
	"172.31.",
	"192.168.",
}

func findWanIP(m map[string]interface{}) string {
	for _, data := range m {
		ports := data.([]interface{})
		port := ports[0].(map[string]interface{})
		ip := port["addr"].(string)
		isPrivate := false
		for _, prefix := range privateIPPrefix {
			if strings.HasPrefix(ip, prefix) {
				isPrivate = true
			}
		}
		if !isPrivate {
			return ip
		}
	}
	return ""
}

func findLanIP(m map[string]interface{}, specificRange string) string {
	for _, data := range m {
		ports := data.([]interface{})
		port := ports[0].(map[string]interface{})
		ip := port["addr"].(string)
		if specificRange != "" {
			if strings.HasPrefix(ip, specificRange) {
				return ip
			}
		} else {
			for _, prefix := range privateIPPrefix {
				if strings.HasPrefix(ip, prefix) {
					return ip
				}
			}
		}
	}
	return ""
}

func showVersion() {
	fmt.Printf("Version: %s\n", Version)
	os.Exit(0)
}
