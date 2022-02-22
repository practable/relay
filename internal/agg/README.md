
![alt text][logo]
# agg
Wrapper around timdrysdale/hub to AGGregate messages from multiple clients


## Usage

Intended as a library for use by timdrysdale/vw


## Definitions

0. Feed: an endpoint that sources/sinks messages e.g. video, audio or experimental data
1. Stream: an aggregrate of messages that are sourced from one or more feeds (e.g. audio and video from a camera)
2. Destination: an endpoint that sources/sinks the messages in a stream (e.g. a data relay for a combined audio/video feed)

Note that streams can NOT work in reverse; i.e. incoming messages from the stream destination are NOT distributed to subClients.

## Operation

A ```timdrysdale/hub``` works by registering client channels to a map of topics. On its own it can connect feeds and destinations, but it has no concept of a stream.  ```timdrysdale/agg``` is intended to provide convenience functions that support working with streams by managing the individual client registrations, and unregistrations, required to dynamically compose and recompose streams that contain multiple feeds. A use case for dynamic stream (re)composition is controlling whether the audio track is sent from the experiment - for privacy reasons, an experimental owner may wish to selectively broadcast the audio track - e.g. turn off audio broadcast when humans are in the same room as the experiment, whilst continuing to send the video frames. This is possible given the nature of MPEG TS streams.

A client can register to a stream as if it were a feed on a ```timdrysdale/hub```. Behind the scenes, the client's message channel will be registered to all the relevant topics.

A client can also unregister to a stream, with the client's message channel being removed from all the relevant topics.

The behaviour upon multiple registrations, is undefined, as is therefore the behaviour on an unregistration of a multiply-registered client. Do not rely on the current implementation's behaviour in this regard - it could change at any time. It is expected that clients only register to each topic once. Any use case requiring message duplication should handle that itself to protect against future changes in the implementation.

When a new rule is received, all clients currently registered to the associated stream have their message channel registered to the appropriate topics.
If the new rule replaces an existing rule, then all clients currently registerd to the stream have their current topic registrations revoked, then they are registered to the new streams. This avoids needing an explicit delete step, and it avoids the implicit state that would otherwise occur if stream rules could be split across multiple 'add'/'delete' commands (which of course, they can't). The number of feeds is expected to be in order of two per stream, so the penalty for needing to fully specify the feeds for each stream is low.

## Rules

Rules for composing streams are simple. Each stream name maps to a list of the constituent feed names. A Rule struct is passed to the addstream channel to create a new stream, or update an existing stream. A Rule struct is passed to the delstream channel to delete the stream. A client registered to a particular stream continues to receive messages according to the latest Rule, hence the composition of the stream can be dynamically altered transparently to the stream client. A stream client registering before a stream rule exists must be connected as soon as a rule is received - this covers off the possibility that rules are deleted then added - in the moment after the rule is deleted and before the new rule is added, the situation is the same as if a client has registered to a non-existent rule.

```go
type Rule struct {
	 Stream string
	 Feeds []string
}
```

So as to avoid circular definitions of streams, which could occur if feeds and streams were not differentiated from each other, streams have their own namespace achieved via prepending or '/stream' to the path, e,g, '/stream/large'. Feeds do not need a namespace, so that behaviour is compatible with ```timdrysdale/hub``` for non-stream usage.



[logo]: ./img/logo.png "AGG logo"