#!/bin/bash

echo 'Building new `autopilot` binary...'
go install

echo 'Installing the plugin...'
cf uninstall-plugin Autopilot
cf install-plugin $GOPATH/bin/autopilot
