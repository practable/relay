#!/bin/bash
export accessport=10000
export relayport=10001
export SHELLTOKEN_LIFETIME=3600
export SHELLTOKEN_ROLE=host
export SHELLTOKEN_SECRET=somesecret
export SHELLTOKEN_TOPIC=123
export SHELLTOKEN_CONNECTIONTYPE=shell
export SHELLTOKEN_AUDIENCE=http://[::]:${accessport}
export host_token=$(shell token)
echo "host_token=${host_token}"
export SHELLHOST_LOCALPORT=22
export SHELLHOST_RELAYSESSION=http://[::]:${accessport}/shell/123
export SHELLHOST_TOKEN=$host_token
export SHELLHOST_DEVELOPMENT=true
shell host
