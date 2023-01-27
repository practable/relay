#!/bin/bash
mkdir ./ignore || true #ignore if fails
export RELAY_ALLOW_NO_BOOKING_ID=true
export RELAY_AUDIENCE=http://localhost
export RELAY_LOG_LEVEL=debug
export RELAY_LOG_FORMAT=text
export RELAY_LOG_FILE=stdout
export RELAY_PORT_ACCESS=3001
export RELAY_PORT_PROFILE=6061
export RELAY_PORT_RELAY=3003
export RELAY_SECRET=$(cat ~/secret/v0/relay.pat)
export RELAY_TIDY_EVERY=5m
export RELAY_URL=ws://localhost:3003
./cmd/relay/relay serve

