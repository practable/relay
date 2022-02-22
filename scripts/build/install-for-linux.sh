#!/bin/bash

# notice how we avoid spaces in $now to avoid quotation hell in go build command
export GOOS=linux
now=$(date +'%Y-%m-%d_%T')
go install -ldflags "-X 'github.com/timdrysdale/gradex-cli/cmd.Version=`git describe`' -X 'github.com/timdrysdale/gradex-cli/cmd.BuildTime=$now'"  

