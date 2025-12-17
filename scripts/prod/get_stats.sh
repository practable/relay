#!/bin/bash
export RELAY_TOKEN_LIFETIME=3600
export RELAY_TOKEN_SCOPE_OTHER=relay:stats
export RELAY_TOKEN_SECRET=$(cat ~/secret/relay.pat) #your location may be different
export RELAY_TOKEN_AUDIENCE=https://test.practable.io/ed0/access #needs the ed0/access on the end
export ACCESS_TOKEN=$(relay token)
curl --header "Authorization: ${ACCESS_TOKEN}" localhost:3000/status
# if remote, then
curl --header "Authorization: ${ACCESS_TOKEN}" test.practable.io/ed0/access/status




