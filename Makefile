VERSION := $(shell go run cmd/monkfish/monkfish.go -version | sed 's/Version: //')
CMDDIR  := ./cmd/monkfish
.PHONY: test setup clean-zip all compress release

monkfish: test
	go build $(CMDDIR)

test:
	go test ./...

setup:
	which gox || go get github.com/mitchellh/gox
	which ghr || go get github.com/tcnksm/ghr

clean-zip:
	find pkg -name '*.zip' | xargs rm

all: setup test
	gox \
	    -os="linux" \
	    -arch="amd64" \
	    -output "pkg/{{.Dir}}_$(VERSION)-{{.OS}}-{{.Arch}}" \
	    $(CMDDIR)

compress: all clean-zip
	cd pkg && ( find . -perm -u+x -type f -name 'monkfish*' | gxargs -i zip -m {}.zip {} )

release: compress
	git push origin master
	ghr -u tech $(VERSION) pkg
	git fetch origin --tags
