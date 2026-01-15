#!/bin/bash
export GOOS=linux
now=$(date +'%Y-%m-%d_%T') #no spaces to prevent quotation issues in build command
(cd ../../cmd/relay-monitor; go build -ldflags "-X 'github.com/practable/relay/cmd/relay-monitor/cmd.Version=`git describe --tags`' -X 'github.com/practable/relay/cmd/relay-monitor/cmd.BuildTime=$now'"; ./relay-monitor version)
