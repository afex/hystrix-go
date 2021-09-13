#!/bin/bash
set -e

wget -q https://storage.googleapis.com/golang/go1.17.1.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.17.1.linux-amd64.tar.gz

apt-get update
apt-get -y install git mercurial apache2-utils gcc # GCC needed for CGO dependencies
apt-get upgrade -y

echo 'export PATH=$PATH:/usr/local/go/bin:/go/bin
export GOPATH=/go' >> /home/vagrant/.profile

source /home/vagrant/.profile

go install golang.org/x/tools/cmd/goimports@latest
go install golang.org/x/lint/golint@latest
go get github.com/smartystreets/goconvey/convey
go get github.com/cactus/go-statsd-client/statsd@v2
go get github.com/rcrowley/go-metrics@latest
go get github.com/DataDog/datadog-go/statsd@latest

chown -R vagrant:vagrant /go

echo "cd /go/src/github.com/afex/hystrix-go" >> /home/vagrant/.bashrc
