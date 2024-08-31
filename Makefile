VERSION    = $(shell git describe --tags --always)
GIT_COMMIT = $(shell git rev-parse --short HEAD)
BUILD_TIME = $(shell date "+%F %T")

define LDFLAGS
"-X 'github.com/eryajf/cloud_dns_exporter/pkg/cmd.Version=${VERSION}' \
 -X 'github.com/eryajf/cloud_dns_exporter/pkg/cmd.GitCommit=${GIT_COMMIT}' \
 -X 'github.com/eryajf/cloud_dns_exporter/pkg/cmd.BuildTime=${BUILD_TIME}'"
endef


default: build

build:
	go build -ldflags=${LDFLAGS} -o cloud_dns_exporter main.go

build-linux:
	CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -ldflags=${LDFLAGS} -o cloud_dns_exporter main.go

lint:
	env GOGC=25 golangci-lint run --fix -j 8 -v ./...