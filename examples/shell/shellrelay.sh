#!/bin/bash
export accessport=10000
export relayport=10001
export SHELLRELAY_ACCESSPORT=$accessport
export SHELLRELAY_ACCESSFQDN=http://[::]:$accessport
export SHELLRELAY_RELAYPORT=$relayport
export SHELLRELAY_RELAYFQDN=ws://[::]:${relayport}
export SHELLRELAY_SECRET=somesecret
export SHELLRELAY_DEVELOPMENT=true
shell relay
