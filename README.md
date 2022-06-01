# Relay

![alt text][logo]

![alt text][status]
![alt text][coverage]

Relay is a set of tools and services to let you to host remote lab experiments, without opening firewall ports.

 - Secure websocket relay and host adapter for sharing video and data, with read/write permissions
 - Secure login shell relay, host adapter and client for end-to-end encrypted admin access without a jumpserver 
 - Booking server for connecting users to experiments
 - Works with experiments behind firewalls and NAT because all communications are relayed 
 - No need to open firewall ports, or get public IPv4 addresses.
 
## Background
 
Relay is the new core of the [practable.io](https://practable.io) remote laboratory ecosystem. Some of the educational thinking behind this ecosystem can be found [here](https://www.tandfonline.com/doi/full/10.1080/23752696.2020.1816845) [1]. 
  
## Status

The system is currently suitable for single-tenacy operations, with a single administrative "zone". Additional automation of experiment and system provision has been developed and will be released once secret-handling has been separated out.

We've successfully used this code to run assessed coursework for over 250 students during Q1/Q2 of 2021, alongside some student recruitment events [2].

We've got over 50 experiments under management at the present time - with some of our latest in our new 1:6 scale ISO containers:

<img src="./img/AGB_Spinners.jpg" width="60%" alt="One-sixth scale model containers holding spinner experiments">

## Overview

This repo contains a system for running experiments behind firewalls, including 

0. `session host` runs on the experiment to connect to the `session relay` to stream data and receive commands
0. `session relay` runs in the cloud to connect experiments and users.
0. `book serve` runs in the cloud (to handle bookings)
0. `shell host` runs on the experiment to connect to the `shell relay` to provide secured `ssh` connections to the experiment
0. `shell relay` runs in the cloud (to connect experiments and administrators)
0. `shell client` runs on the administrators' systems to connect to the `shell relay` 

<figure>
<img src="./img/data-flow-diagram.svg" width="100%" alt="data flow diagram of the relay system">
<figcaption align = "center"><b>Dataflow diagram of the `session host` to `session relay` connection, reproduced from [2] under CC-BY-4.0 license</b></figcaption>
</figure>

### More information

Additonal documentation (in various states of completeness) can be found on the following components here:

0. [booking](./internal/booking/README.md)
0. [booking client](./internal/bc/README.md)
0. [session](./cmd/session/README.md)
0. [shell relay](./internal/shellrelay/README.md)
0. [shell host](./internal/shellhost/README.md)

## References

[1] Timothy D. Drysdale (corresponding author), Simon Kelley, Anne-Marie Scott, Victoria Dishon, Andrew Weightman, Richard James Lewis & Stephen Watts (2020) Opinion piece: non-traditional practical work for traditional campuses, Higher Education Pedagogies, 5:1, 210-222, DOI: 10.1080/23752696.2020.1816845 

[2] David P. Reid, Joshua Burridge, David B. Lowe, and Timothy D. Drysdale (corresponding author), Open-source remote laboratory experiments for controls engineering education, International Journal of Mechanical Engineering Education, Accepted 22 Jan 2022. 


[status]: https://img.shields.io/badge/status-operating-green "status; development"
[coverage]: https://img.shields.io/badge/coverage-44%25-orange "Test coverage 44%"
[logo]: ./img/logo.png "Relay ecosystem logo - hexagons connected in a network to a letter R"


## Appendix

In combining the commands, it would be helpful to consolidate the environment variables needed

### Original list of environment variables

There are approx 85 listed at present.

`grep -r 'export' -r | grep -v README | grep -v '~' | sed 's/.*:export//'`

```
 BOOKRESET_TOKEN=${your_admin_login_token}
 BOOKRESET_HOST=localhost
_BOOKRESET_SCHEME=http
 BOOKTOKEN_LIFETIME=300
 BOOKTOKEN_SECRET=somesecret
 BOOKTOKEN_ADMIN=true
 BOOKTOKEN_AUDIENCE=https://book.example.io
 BOOKTOKEN_GROUPS="group1 group2 group3"
 BOOKSTATUS_BASE=/book/api/v1
 BOOKSTATUS_HOST=core.prac.io
 BOOKSTATUS_SCHEME=https
 BOOKSTATUS_TOKEN=$secret
 BOOKUPLOAD_TOKEN=${your_admin_login_token}
_BOOKUPLOAD_SCHEME=https
 BOOKUPLOAD_HOST=core.prac.io
 BOOKUPLOAD_BASE=/book/api/v1
 BOOKSTATUS_SCHEME=https
 BOOKSTATUS_HOST=core.prac.io
 BOOKSTATUS_BASE=/book/api/v1
 BOOKSTATUS_TOKEN=$secret
 BOOK_PORT=4000
 BOOK_FQDN=https://book.practable.io
 BOOK_LOGINTIME=3600
 BOOK_SECRET=somesecret
 ACCESSTOKEN_LIFETIME=86400
 ACCESSTOKEN_ROLE=client
 ACCESSTOKEN_SECRET=$($HOME/secret/session_secret.sh)
 ACCESSTOKEN_TOPIC=spin35-data
 ACCESSTOKEN_CONNECTIONTYPE=session
 ACCESSTOKEN_AUDIENCE=https://relay-access.practable.io
 SESSION_CLIENT_TOKEN=$(session token)
 SESSION_CLIENT_FILE_DEVELOPMENT=true
 SESSION_CLIENT_SESSION=$ACCESSTOKEN_AUDIENCE/$ACCESSTOKEN_CONNECTIONTYPE/$ACCESSTOKEN_TOPIC
 SESSION_CLIENT_FILE_LOG=/var/log/session/spin35-data.log
 SESSION_CLIENT_SESSION=$ACCESSTOKEN_AUDIENCE/$ACCESSTOKEN_CONNECTIONTYPE/$ACCESSTOKEN_TOPIC
 SESSION_CLIENT_FILE_LOG=/var/log/session/spin35-data.log
 SESSION_CLIENT_FILE_PLAY=/etc/practable/spin35-check.play
 SESSION_CLIENT_FILE_INTERVAL=10ms
 SESSION_CLIENT_FILE_FORCE=true
 SESSION_CLIENT_FILE_CHECK_ONLY=true
 SESSION_CLIENT_FILE_PLAY=/etc/practable/spin35-check.play
 pid=$!
 RELAYHOST_PORT=8888
 RELAYHOST_LOGLEVEL=PANIC
 RELAYHOST_MAXBUFFERLENGTH=10
 RELAYHOST_CLIENTBUFFERLENGTH=5
 RELAYHOST_CLIENTTIMEOUTMS=1000
 RELAYHOST_HTTPWAITMS=5000
 RELAYHOST_HTTPSFLUSHMS=5
 RELAYHOST_HTTPTIMEOUTMS=1000
 RELAYHOST_CPUPROFULE=
 RELAYHOST_API=
 RELAY_ACCESSPORT=10002
 RELAY_ACCESSFQDN=https://access.example.io
 RELAY_RELAYPORT=10003
 RELAY_RELAYFQDN=wss://relay-access.example.io
 RELAY_SECRET=somesecret
 RELAY_DEVELOPMENT=true
 SESSION_CLIENT_SESSION=https://relay-access.practable.io/session/govn05-data
 SESSION_CLIENT_TOKEN=ey... #include complete JWT token
 ACCESSTOKEN_LIFETIME=3600
 ACCESSTOKEN_READ=true
 ACCESSTOKEN_WRITE=true
 ACCESSTOKEN_SECRET=somesecret
 ACCESSTOKEN_TOPIC=123
 ACCESSTOKEN_AUDIENCE=https://relay-access.example.io
 SHELLHOST_LOCALPORT=22
 SHELLHOST_RELAYSESSION=https://access.example.io/shell/abc123
 SHELLHOST_TOKEN=ey...<snip>
 SHELLHOST_DEVELOPMENT=true
 SHELLRELAY_ACCESSPORT=10001
 SHELLRELAY_ACCESSFQDN=https://access.example.io
 SHELLRELAY_RELAYPORT=10000
 SHELLRELAY_RELAYFQDN=wss://relay-access.example.io
 SHELLRELAY_SECRET=$your_secret
 SHELLRELAY_DEVELOPMENT=true
 SHELLCLIENT_LOCALPORT=22
 SHELLCLIENT_RELAYSESSION=https://access.example.io/shell/abc123
 SHELLCLIENT_TOKEN=ey...<snip>
 SHELLCLIENT_DEVELOPMENT=true
 SHELLTOKEN_LIFETIME=3600
 SHELLTOKEN_ROLE=client
 SHELLTOKEN_SECRET=somesecret
 SHELLTOKEN_TOPIC=123
 SHELLTOKEN_CONNECTIONTYPE=shell
 SHELLTOKEN_AUDIENCE=https://shell-access.example.io
```

## Reduced list

There are less than 40 unique environment variables needed
 
```
#generic
RELAY_DEVELOPMENT
RELAY_LOGLEVEL
RELAY_CPUPROFILE #not needed, as mispelt?

# for running a command
RELAY_TOKEN
RELAY_HOST
RELAY_SCHEME
RELAY_BASE
RELAY_SESSION #for client

# for setting a token
RELAY_LIFETIME
RELAY_SECRET
RELAY_AUDIENCE

# specific to particular types
RELAY_TOPIC
RELAY_CONNECTION_TYPE
RELAY_GROUPS
RELAY_ADMIN
RELAY_ROLE
RELAY_READ
RELAY_WRITE

# for serving something
RELAY_ACCESS_FQDN
RELAY_ACCESS_PORT
RELAY_RELAY_FQDN
RELAY_RELAY_PORT
RELAY_LOGIN_TIME_S #for book

#for relay host
RELAY_MAX_BUFFER_LENGTH
RELAY_CLIENT_BUFFER_LENGTH
RELAY_CLIENT_TIMEOUT_MS
RELAY_HTTP_WAIT_MS
RELAY_HTTP_FLUSH_MS
RELAY_HTTP_TIMEOUT_MS



# stuff to do with session client
RELAY_LOG
RELAY_PLAY
RELAY_INTERVAL
RELAY_FORCE
RELAY_CHECK_ONLY

```
