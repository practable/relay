#!/bin/bash
export GOOS=linux
now=$(date +'%Y-%m-%d_%T') #no spaces to prevent quotation issues in build command
(cd ../../cmd/relay; go build -ldflags "-X 'github.com/practable/relay/cmd/relay/cmd.Version=`git describe --tags`' -X 'github.com/practable/relay/cmd/relay/cmd.BuildTime=$now'"; ./relay version)
