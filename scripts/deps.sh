#!/bin/sh

set -e

rm -rf Godeps
go get -u -f -t ./...
godep save ./... github.com/onsi/ginkgo/ginkgo
