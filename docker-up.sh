#!/bin/bash

cat << EOF > relay.env
RELAY_ACCESSPORT=10002
RELAY_ACCESSFQDN=https://relay-access.practable.io
RELAY_ALLOWNOBOOKINGID=true
RELAY_RELAYPORT=10003
RELAY_RELAYFQDN=wss://relay.practable.io
RELAY_SECRET=$(cat ~/secret/sessionrelay.pat)
RELAY_DEVELOPMENT=true
RELAY_PRUNEEVERY=5m 
EOF

docker-compose up

