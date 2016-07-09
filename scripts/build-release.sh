#!/bin/bash

set -e

SANDBOX=$(mktemp -d)

echo "building linux..."
GOOS=linux go build -o $SANDBOX/autopilot-linux github.com/contraband/autopilot

echo "building os x..."
GOOS=darwin go build -o $SANDBOX/autopilot-darwin github.com/contraband/autopilot

echo "building windows..."
GOOS=windows go build -o $SANDBOX/autopilot.exe github.com/contraband/autopilot

echo

find $SANDBOX -type f -exec file {} \;

echo
echo "binaries are in $SANDBOX!"
