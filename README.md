# monkfish

[![monkfish](https://upload.wikimedia.org/wikipedia/commons/thumb/b/b0/FMIB_46047_Monkfish.jpeg/800px-FMIB_46047_Monkfish.jpeg)](https://commons.wikimedia.org/wiki/File:FMIB_46047_Monkfish.jpeg)

Generate the Hosts Dynamically on OpenStack servers

## How to install

```
$ go get github.com/udzura/monkfish/cmd/monkfish

# Or on project root
$ make install
```

### Linux amd64 binary is

* Here: https://github.com/udzura/monkfish/releases/latest

## Usage

```
$ monkfish -help 
Usage of monkfish:
  -V    Verbose mode
  -c string
        Config path (default "/etc/monkfish.ini")
  -t string
        Target file to write hosts (default "/etc/hosts")
  -version
        Just show version and quit
  -w    Write to file
```

## Just keep it in crontab

First, set the `/etc/hosts.base` e.g. running `sudo cp -a /etc/hosts /etc/hosts.base`

Then, create a config file `/etc/monkfish.ini` like:

```
[default]
os_username = "udzura"
os_password = "t0nkotsu-r@men"
os_tenant_name = "the_tenant"
os_auth_url = "https://your.keystone.host:9999/v2.0"
os_region = "RegionOne"
domain = "monk.example.tld"
internal_domain = "monk.local"
lan_ip_prefix = "10.10.100." # Optional
```

Then set crontab:

```crontab
@reboot     root /usr/local/bin/monkfish -w
*/3 * * * * root /usr/local/bin/moknfish -w
```

[Pro tips] When you want to avoid thundering herd:

```crontab
*/3 * * * * bash -c 'sleep $(($RANDOM \% 60)) && /usr/local/bin/moknfish -w'
```

After this the `/etc/hosts` will be periodicaly updated from `/etc/hosts.base` and existing server networks.

```
127.0.0.1   localhost localhost.localdomain localhost4 localhost4.localdomain4
::1         localhost localhost.localdomain localhost6 localhost6.localdomain6


54.248.999.999          proxy001.monk.example.tld
10.187.100.10           proxy001.monk.local
176.34.999.999          worker003.monk.example.tld
10.187.100.11           worker003.monk.local
175.41.999.999          worker004.monk.example.tld
10.187.100.12           worker004.monk.local
...
```

## Testing that it really works

* Do not pass `-w` if you don't want to update hosts. Without `-w`, entries are outputted to stdout.
* Pass `-V` for debug.

```
/usr/local/bin/monkfish -V

name: proxy001
54.248.999.999          proxy001.monk.example.tld
10.187.100.10           proxy001.monk.local
name: worker003
176.34.999.999          worker003.monk.example.tld
10.187.100.11           worker003.monk.local
...
```

## Special Thanks

Original idea is from the implementation of @lamanotrama's Perl script.

## License

[MIT](./LICENSE).

## Roadmap

* Automated update via `consul watch`
