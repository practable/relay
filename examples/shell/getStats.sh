#!/bin/sh

# Make token
export accessport=10000
export SHELLTOKEN_LIFETIME=30
export SHELLTOKEN_ROLE=stats
export SHELLTOKEN_SECRET=somesecret
export SHELLTOKEN_TOPIC=stats
export SHELLTOKEN_CONNECTIONTYPE=shell
export SHELLTOKEN_AUDIENCE=http://[::]:${accessport}
export client_token=$(shell token)
echo "client_token=${client_token}"

# Request Access
export ACCESS_URL=http://localhost:${accessport}/shell/stats

export STATS_URL=$(curl -X POST  \
-H "Authorization: ${client_token}" \
$ACCESS_URL | jq -r '.uri')

echo $STATS_URL
# Connect to stats channel & issue {"cmd":"update"}

echo '{"cmd":"update"}' | websocat -n1 "$STATS_URL" | jq .
