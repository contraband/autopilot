#!/bin/bash

set -e

SANDBOX=$(mktemp -d)

echo "building linux..."
CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go build -o $SANDBOX/autopilot-linux github.com/contraband/autopilot

echo "building os x..."
CGO_ENABLED=0 GOARCH=amd64 GOOS=darwin go build -o $SANDBOX/autopilot-darwin github.com/contraband/autopilot

echo "building windows..."
CGO_ENABLED=0 GOARCH=amd64 GOOS=windows go build -o $SANDBOX/autopilot.exe github.com/contraband/autopilot

echo

find $SANDBOX -type f -exec file {} \;

echo
echo "binaries are in $SANDBOX!"
