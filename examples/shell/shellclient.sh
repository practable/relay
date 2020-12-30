#!/bin/bash
export accessport=10000
export secret=somesecret
export SHELLTOKEN_LIFETIME=3600
export SHELLTOKEN_ROLE=client
export SHELLTOKEN_SECRET=somesecret
export SHELLTOKEN_TOPIC=123
export SHELLTOKEN_CONNECTIONTYPE=shell
export SHELLTOKEN_AUDIENCE=http://[::]:${accessport}
export client_token=$(shell token)
echo "client_token=${client_token}"
export SHELLCLIENT_LOCALPORT=2222
export SHELLCLIENT_RELAYSESSION=http://[::]:${accessport}/shell/123
export SHELLCLIENT_TOKEN=$client_token
export SHELLCLIENT_DEVELOPMENT=true
shell client

