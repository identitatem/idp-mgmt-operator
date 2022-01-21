#!/bin/bash

# Copyright Red Hat

set -e

# Install go
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
if [[ "$ARCH" == "x86_64" ]]; then
    ARCH="amd64"
fi

mkdir -p /go/{bin,pkg,src} /go/pkg/cache /opt/go
chmod -R a+rwx /go
cd /opt/go
wget --progress=dot:mega https://dl.google.com/go/go${GOVERSION}.${OS}-${ARCH}.tar.gz
tar -C /usr/local -xzf go${GOVERSION}.${OS}-${ARCH}.tar.gz
rm -rf /opt/go
if [ "$ARCH" = "s390x" ]; then
        cd /usr/bin
        ln gcc s390x-linux-gnu-gcc
fi
