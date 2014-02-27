#!/bin/sh
set -e
apt-get update
apt-get install -y git
gem install librarian-puppet

mkdir -p /etc/puppet
cp /go/src/github.com/afex/hystrix-go/puppet/Puppetfile /etc/puppet

cd /etc/puppet && librarian-puppet install --clean --verbose
