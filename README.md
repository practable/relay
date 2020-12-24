# relay
Secure websocket relay server and clients for sharing video, data, and ssh across firewall boundaries


IN DEVELOPMENT NOW - REPO HERE FOR BACKUP ONLY - DO NOT USE - API IN FLUX


Ignore below here (README.md from crossbar ...)


```










#Crossbar
![alt text][logo]
![alt text][status]
![alt text][coverage]
```
Crossbar relays websocket streams

## Why?

Remote laboratories require a custom video and data communications eco-system in order to support their wider adoption. Key factors include:

+ Remote laboratory participants (whether human or machine) are often located behind institutional firewalls and NAT
+ Most instituational networks support particpants sending and receiving video and data to external relays, but not acting as a server
+ Those data streams are typically embedded in websockets, whereas UDP and some TCP protocols are sometimes explicitly blocked
+ Almost all video and data messaging vendors with relays are focused solely on human-human communications
    + often missing apparently-minor features from the API that are key for remote lab experiments (e.g. being able to change camera programmatically)
	+ often require workarounds for remote lab adminstration tasks which are prevented by privacy features in browsers (e.g. identifying cameras)
	+ typically require x10 more expensive computer for the experiment because of the overhead of running graphical operating system and browser
	+ most vendors can - quite rightly - only guarantee long-term support for users that conform to their core use-case

## Features

+ binary and text websockets
+ multiple, independent streams
    + organised by topics
	+ topics set by routing
+ multiple streaming schemas	
    + bidirectional N:N streaming
    + unidirectional 1:N streaming
+ streaming schemas are 
    + set by routing
	+ individually selectable for each stream 
+ statistics are recorded, and available via
    + websocket API in JSON format
	+ human-readable webpage with 5sec update at ```<host>/stats```
	
## What's with the name?
I once had an old IBM p690 compute cluster whose processor cores had crossbar switches ([Core Interface Units](http://www.netlib.org/utk/papers/advanced-computers/power4.html)) that connect any core with any cache, and [do so more efficiently than a standard compute cluster](http://www.piers.org/piersonline/pdf/Vol5No2Page117to120.pdf). It seemed apt, because this relay is about connecting any experiment with any human, more efficiently than existing systems, (in a holistic sense, including total cost and effort of ownership, administration, maintenance etc).

## Performance

After some profiling-led optimisation, I've run the code for four months continuously (as of March 2020) with multiple streams and re-connections of broadcasters and receivers, including a fortnight where a stream was connecting to a new routing every second (in a sequence of four) using a bash script and [timdrysdale/vw](https://github.com/timdrysdale/vw), with no appreciable leakage of memory.

Further quantitative benchmarking is needed to understand how well crossbar scales to handle large numbers of streams and clients simultaneously, but from the perspective of a light user with around four video streams being in use for live demonstrations at any one time, it has performed well and given confidence. Performance at low N is no guarantee of performance at high N, so please conduct your own testing if adopting at this time (and share with me for inclusion here, via email or PR). 

Meanwhile, here's a text-grab from ```top``` on an Amazon EC2 ```c5.Large``` instance showing the same memory usage now as four months ago under the same load.

```
top - 12:57:15 up 232 days, 22:45, 14 users,  load average: 0.04, 0.04, 0.00
Tasks: 123 total,   1 running,  83 sleeping,   0 stopped,   0 zombie
%Cpu(s):  2.0 us,  0.5 sy,  0.0 ni, 97.5 id,  0.0 wa,  0.0 hi,  0.0 si,  0.0 st
KiB Mem :  3794284 total,   486436 free,   691080 used,  2616768 buff/cache
KiB Swap:        0 total,        0 free,        0 used.  2831916 avail Mem 

  PID USER      PR  NI    VIRT    RES    SHR S  %CPU %MEM     TIME+ COMMAND             
24363 root      20   0  252616  52060   4720 S   4.0  1.4   1019:36 iftop               
31302 ubuntu    20   0  930664  26504   8856 S   1.7  0.7   3852:37 crossbar            
 5917 www-data  20   0  145712  11396   7852 S   1.3  0.3 123:49.74 nginx               
23179 root      20   0  447328  42808  23328 S   0.7  1.1 292:00.12 containerd          
  816 jvb       20   0 5770988 135244   8936 S   0.3  3.6   1002:19 java                
23051 root      20   0  471992  76156  40564 S   0.3  2.0 181:01.84 dockerd             
    1 root      20   0  225484   9556   7068 S   0.0  0.3  12:45.70 systemd
```

## Getting started

Binaries will be available in the future, but for now it is straightforward to compile.

It's quick and easy to create a working ```go``` system, [following the advice here](https://golang.org/doc/install).

Get the code, test, and install
```
$ go get github.com/timdrysdale/crossbar
$ cd `go list -f '{{.Dir}}' github.com/timdrysdale/crossbar`
$ go test ./cmd
$ go install
```
To run the relay
```
$export CROSSBAR_LISTEN=127.0.0.1:9999
$ crossbar
```

Navigate to the stats page, e.g. ```http://localhost:9999/stats```

You won't see any connections, yet.

You can connect using any of the other tools in the ```practable``` ecosystem, e.g. [timdrysdale/vw](https://github.com/vw/timdrysdale) or if you already have the useful [websocat](https://github.com/vi/websocat) installed, then

```
websocat --text ws://127.0.0.1:8089/expt - 
```
If you type some messages, you will see the contents of the row change to reflect that you have sent messages.


### Multiple clients are supported

If you connect a second or even third or more times from other terminals, you will see the hub relaying your messages to all other clients.
Try typing in each of the terminals, and see that your message makes it to each of the others.

### Streams are independent

Try setting up a pair of terminals that are using a different topic, and notice that messages do not pass from one topic to another.

e.g. connect from two terminals using
```
websocat --text ws://127.0.0.1:8089/sometopic - 
```

Messages sent in a terminal connected to ```<>/sometopic``` will only go to terminals connected to the same route, and not to any other terminal.

Here's a screenshot (note that ```websocat``` does a local echo so the sender can see their message; the echo is not from ```crossbar```)

![alt text][topics]

## Usage examples

The default is bidirectional messaging, as you have seen in the example above. 

### Unidirectional messaging

If you only want to broadcast messages, such as a video stream, then it is nice to have some certainty that one of your clients won't inadvertently mess up the video for others by transmitting some sort of reply. To take advantage of uni-directional messaging, start the server's route with ```/in/``` and the clients' routes with ```/out/```. Note that the rest of the route has to match.

You can try it out yourself.

```
websocat --text ws://127.0.0.1:8089/in/demo - 
```
and
```
websocat --text ws://127.0.0.1:8089/out/demo - 
```

You can see from the local echo that messages attempted to be sent from the clients connected to ```/out/``` are not sent to any other clients - this is enforced by the hub and does not need any special behaviour from the clients (beyond connecting to the right route). Protecting unauthorised users from connecting to the ```/in/``` route is outside the scope of the ```crossbar``` codebase, in line with conventional practice on separating concerns. 

![alt text][unidirectional]

## Applications

### Relaying video and data

Crossbar has been successfully relaying MPEG video and JSON data (on separate topics) for [penduino-ui](https://github.com/timdrysdale/penduino-ui) experiments using [ffmpeg](https://ffmpeg.org) and [timdrysdale/vw](https://github.com/timdrysdale/vw).  

### Relaying shell access

Shell relay is in a separate project because these have different goals, development schedules, and performance targets, even though some of the underlying code and approach is similar. See [timdrysdale/shellbar]( https://github.com/timdrysdale/shellbar).

## Support / Contributions

Get in touch! My efforts are going into system development at present so support for alpha-stage users is by personal arrangement/consultancy at present. If you wish to contribute, or have development resource available to do so, I'd be more than delighted to discuss with you the possibilities.

## Developer stuff

The swagger2.0 spec for the relay-access API can be used to auto-generate the boilerplate for the server code with the command
```
swagger generate server -t pkg/access -f ./api/openapi-spec/access.yml --exclude-main -A access
```

Some links to articles on swagger and authentication: [jwt auth](https://shashankvivek-7.medium.com/go-swagger-user-authentication-securing-api-using-jwt-part-2-c80fdc1a020a); [context](https://medium.com/@cep21/how-to-correctly-use-context-context-in-go-1-7-8f2c0fafdf39).


[status]: https://img.shields.io/badge/alpha-working-green "Alpha status; working"
[coverage]: https://img.shields.io/badge/coverage-54%25-orange "Test coverage 54%"
[logo]: ./img/logo.png "Crossbar logo - hexagons connected in a network to a letter C"
[topics]: ./img/topics.png "Multiple terminals showing bidirectional message flow"
[unidirectional]: ./img/unidirectional.png "Multiple terminals showing unidirectional message flow"
