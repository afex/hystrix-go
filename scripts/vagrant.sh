#!/bin/bash
set -e

wget https://go.googlecode.com/files/go1.2.1.linux-amd64.tar.gz
tar -C /usr/local -xzf go1.2.1.linux-amd64.tar.gz

apt-get -y install git mercurial

echo 'export PATH=$PATH:/usr/local/go/bin:/go/bin
export GOPATH=/go' >> /home/vagrant/.profile

source /home/vagrant/.profile

go get code.google.com/p/go.tools/cmd/goimports
go get github.com/golang/lint/golint

chown -R vagrant:vagrant /go
