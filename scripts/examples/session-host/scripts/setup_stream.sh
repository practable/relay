#!/bin/sh 
curl -X POST -H "Content-Type: application/json" -d '{"stream":"/stream/front/large","destination":"wss://video.practable.io:443/in/video1","id":"0"}' http://localhost:8888/api/destinations
curl -X POST -H "Content-Type: application/json" -d '{"stream":"/stream/front/large","feeds":["video0","audio0"]}' http://localhost:8888/api/streams
sleep 15s
curl -X GET -H "Content-Type: application/json" http://localhost:8888/api/destinations/all
curl -X DELETE "Content-Type: application/json" http://localhost:8888/api/destinations/0
