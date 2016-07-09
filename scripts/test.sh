#!/bin/bash
# vim: set ft=sh

set -e

export GOPATH=$PWD/gopath
export PATH=$GOPATH/bin:$PATH

cd $GOPATH/src/github.com/contraband/autopilot

export GOPATH=${PWD}/Godeps/_workspace:$GOPATH
export PATH=${PWD}/Godeps/_workspace/bin:$PATH

go install github.com/onsi/ginkgo/ginkgo

ginkgo -r "$@"
