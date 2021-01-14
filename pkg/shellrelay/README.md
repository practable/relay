# Shellrelay

Shellrelay connects clients with remote sshd behind firewalls, by relaying the connection. This is intended for use cases where an incoming ssh connection would be acceptable, but the administration in arranging the incoming ports open on the firewall is burdensome.

## Known issues 

Some infrequent data corruption observed during moderate traffic (using top) may be down to framing errors in the tcp code, and this could be be diagnosed by comparing against the websocat implementation as a front end, as part of exploring `packet` options.

https://github.com/vi/websocat/issues/42

> Yes. How do you want to contact this daemon? Over TCP or UNIX socket?
>  
> For example,
> 
> `websocat -Et tcp-l:127.0.0.1:1234 reuse-raw:ws://echo.websocket.org`
> acts as a server, forwarding multiple incoming TCP connections to one reused WebSocket connection. Each line of incoming data is transformed to a WebSocket message, each incoming WebSocket message is transformed into a line, which is sent to some currently connected client (or cached up and sent to subsequent client if current client suddenly disconnected).
> `echo 'request in JSON' | nc -q 1 127.0.0.1 1234 > reply.json`
> acts as a client (you can also use something like websocat -b - tcp:127.0.0.1:1234 as a TCP client, but there may be some issues).


## SSH protocol speaking order

One of the quirks of dynamically creating unique connections is that in practice, clients speak their identification code as soon as they connect. This "normal" unrelayed connection was captured with the wireshark display filter "ssh", and aborted at the password prompt:

```
No.     Time           Source                Destination           Protocol Length Info
    154 11.163056509   127.0.0.1             127.0.0.1             SSHv2    109    Client: Protocol (SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.1)
    156 11.184487361   127.0.0.1             127.0.0.1             SSHv2    109    Server: Protocol (SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.1)
    158 11.185665156   127.0.0.1             127.0.0.1             SSHv2    1580   Client: Key Exchange Init
    160 11.187280805   127.0.0.1             127.0.0.1             SSHv2    1124   Server: Key Exchange Init
    162 11.195782883   127.0.0.1             127.0.0.1             SSHv2    116    Client: Diffie-Hellman Key Exchange Init
    164 11.212916150   127.0.0.1             127.0.0.1             SSHv2    576    Server: Diffie-Hellman Key Exchange Reply, New Keys, Encrypted packet (len=228)
    166 11.223024721   127.0.0.1             127.0.0.1             SSHv2    84     Client: New Keys
    168 11.224320095   127.0.0.1             127.0.0.1             SSHv2    112    Client: Encrypted packet (len=44)
    170 11.224457595   127.0.0.1             127.0.0.1             SSHv2    112    Server: Encrypted packet (len=44)
    172 11.224516734   127.0.0.1             127.0.0.1             SSHv2    128    Client: Encrypted packet (len=60)
    174 11.233619762   127.0.0.1             127.0.0.1             SSHv2    120    Server: Encrypted packet (len=52)
    176 11.233845670   127.0.0.1             127.0.0.1             SSHv2    688    Client: Encrypted packet (len=620)
    178 11.242935276   127.0.0.1             127.0.0.1             SSHv2    120    Server: Encrypted packet (len=52)
    179 11.243136681   127.0.0.1             127.0.0.1             SSHv2    432    Client: Encrypted packet (len=364)
    181 11.252221964   127.0.0.1             127.0.0.1             SSHv2    120    Server: Encrypted packet (len=52)

```
With an earlier version of shellbar, that did not enforce server speaks first, the client's initial identification message was lost, and the server dropped the connection because the order of events was wrong. Other has noted similar issues when relaying SSH without a wrapper via nginx, with [ssl_preread enabled](https://unix.stackexchange.com/questions/590602/ssh-connection-issues-from-5-3-client-to-7-4-server). That's not the case here, but the outcome was similar :-


```
No.     Time           Source                Destination           Protocol Length Info
    133 4.197317465    127.0.0.1             127.0.0.1             SSH      109    Server: Protocol (SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.1)
    147 4.203036322    127.0.0.1             127.0.0.1             SSH      1580   Client: Encrypted packet (len=1512)
    149 4.203445448    127.0.0.1             127.0.0.1             SSH      102    Server: Encrypted packet (len=34)
    151 4.203494532    127.0.0.1             127.0.0.1             SSH      70     Server: Encrypted packet (len=2)
```

It's clear that the client is rushing ahead with sending the identification string, probably on the basis that you'd never know whether it had waited for the server's string anyway, so it is a time saving which doesn't hurt any one under normal conditions.

The shellbar hub has been fixed to pause sending of messages from clients until the server has connected. Client messages back up in the channel buffer, and are released when that client receives a write from the server. After applying this fix in commit 7dc5b616848c473bb12016bf2682f1ace76c2098, the correct events now occur, and the formal order has been restored - this is not needed, just confirmation that the client messages are being held back until the server is connected

```
No.     Time           Source                Destination           Protocol Length Info
    299 16.727279145   127.0.0.1             127.0.0.1             SSHv2    109    Server: Protocol (SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.1)
    309 16.730213237   127.0.0.1             127.0.0.1             SSHv2    109    Client: Protocol (SSH-2.0-OpenSSH_8.2p1 Ubuntu-4ubuntu0.1)
    317 16.732642135   127.0.0.1             127.0.0.1             SSHv2    1580   Client: Key Exchange Init
    319 16.733305154   127.0.0.1             127.0.0.1             SSHv2    1124   Server: Key Exchange Init
    333 16.741506715   127.0.0.1             127.0.0.1             SSHv2    116    Client: Diffie-Hellman Key Exchange Init
    336 16.757211929   127.0.0.1             127.0.0.1             SSHv2    576    Server: Diffie-Hellman Key Exchange Reply, New Keys, Encrypted packet (len=228)
    352 16.773040330   127.0.0.1             127.0.0.1             SSHv2    128    Client: New Keys, Encrypted packet (len=44)
    354 16.773215137   127.0.0.1             127.0.0.1             SSHv2    112    Server: Encrypted packet (len=44)
    363 16.777081801   127.0.0.1             127.0.0.1             SSHv2    128    Client: Encrypted packet (len=60)
    364 16.786193261   127.0.0.1             127.0.0.1             SSHv2    120    Server: Encrypted packet (len=52)
    373 16.790738976   127.0.0.1             127.0.0.1             SSHv2    688    Client: Encrypted packet (len=620)
    374 16.799848884   127.0.0.1             127.0.0.1             SSHv2    120    Server: Encrypted packet (len=52)
    382 16.805513533   127.0.0.1             127.0.0.1             SSHv2    432    Client: Encrypted packet (len=364)
    383 16.814736506   127.0.0.1             127.0.0.1             SSHv2    120    Server: Encrypted packet (len=52)
```

The operation has been checked by running the scripts in 'examples/shell' in separate terminals, starting with `shellrelay.sh` then `shellhost.sh` then `shellclient.sh`

Connection is made with 
```
ssh $USER@localhost -p2222
```

Quick manual testing with three concurrent session showing running `top` work just fine.

Note that the token lifetime is limited to an hour in this example, so the connection will be closed after 60 minutes.

