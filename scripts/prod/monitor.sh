#!/bin/bash
export RELAY_MONITOR_AUDIENCE="${HTTPS_HOST}/access"
export RELAY_MONITOR_SECRET=$(<${SECRETS}/relay.pat)
export RELAY_MONITOR_LOG_LEVEL=debug
export RELAY_MONITOR_LOG_FORMAT=json
export RELAY_MONITOR_THRESHOLD=50ms
export RELAY_MONITOR_INTERVAL=5s
export RELAY_MONITOR_NO_RETRIGGER_WITHIN=1m
export RELAY_MONITOR_TRIGGER_AFTER_MISSES=10
export RELAY_MONITOR_TOPIC=canary-st-data
export RELAY_MONITOR_COMMAND=echo "Relay monitor triggered at \$(date)"
# in production, this might be a script that sends an alert
# or kills the relay process for systemd to restart it
# you can't just pkill relay, because that would also kill the monitor
#export RELAY_MONITOR_COMMAND=kill -9 $(pgrep -f "relay serve"))
relay monitor 