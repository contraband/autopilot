#!/bin/bash
# vim: set ft=sh

set -e

export GOPATH=$PWD/gopath
export PATH=$GOPATH/bin:$PATH

cd $GOPATH/src/github.com/contraband/autopilot

go install github.com/contraband/autopilot/vendor/github.com/onsi/ginkgo/ginkgo

ginkgo -r "$@"
