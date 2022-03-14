#!/bin/bash

# example ./log-only.sh 86400 spin35-data $HOME/tmp

# Make token 
export ACCESSTOKEN_LIFETIME=$1
export ACCESSTOKEN_ROLE=client
export ACCESSTOKEN_SECRET=$($HOME/secret/session_secret.sh)
export ACCESSTOKEN_TOPIC=$2
export ACCESSTOKEN_CONNECTIONTYPE=session
export ACCESSTOKEN_AUDIENCE=https://relay-access.practable.io
export SESSION_CLIENT_TOKEN=$(session token)

# set up other options
export SESSION_CLIENT_SESSION=$ACCESSTOKEN_AUDIENCE/$ACCESSTOKEN_CONNECTIONTYPE/$ACCESSTOKEN_TOPIC
mkdir -p $3
export SESSION_CLIENT_FILE_LOG=$3/session.log
export SESSION_CLIENT_FILE_DEVELOPMENT=true
../../../cmd/session/session client file
