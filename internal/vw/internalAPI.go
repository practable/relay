package vw

import (
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/practable/relay/internal/agg"
	"github.com/practable/relay/internal/hub"
	"github.com/practable/relay/internal/rwc"
	log "github.com/sirupsen/logrus"
)

func (app *App) internalAPI(topic string) {

	c := &hub.Client{Hub: app.Hub.Hub,
		Name:  "admin",
		Send:  make(chan hub.Message),
		Topic: topic,
	}

	app.Hub.Register <- c

	for {
		select {
		case message, ok := <-c.Send:

			if !ok {
				// The hub closed the channel.
				return
			}

			reply, err := app.handleAdminMessage(message.Data)

			if err == nil {
				c.Hub.Broadcast <- hub.Message{Sender: *c, Data: reply, Type: websocket.TextMessage, Sent: time.Now()} //mmmm type needed here == too much coupling ...!!
			} else {
				c.Hub.Broadcast <- hub.Message{Sender: *c, Data: []byte(`{"error":"` + err.Error() + `"}`), Type: websocket.TextMessage, Sent: time.Now()}
			}

		case <-app.Closed:
			return
		}
	}
}

// Command represents a command and associated rule
type Command struct {
	Verb  string
	What  string
	Which string
	Rule  *json.RawMessage
}

// RuleStream represents a rule for a stream
type RuleStream struct {
	Stream string
	Feeds  []string
}

var errBadCommand = errors.New("unrecognised Command")
var errNoDeleteAPIRule = errors.New("cannot delete apiRule")

// JSON API - note change to singular stream and destination
//
// {"verb":"add","what":"destination","rule":{"stream":"video0","destination":"wss://<some.relay.server>/in/video0","id":"0"}}
// {"verb":"add","what":"stream","rule":{"stream":"video0","feeds":["video0","audio0"]}}
//
// {"verb":"list","what":"stream","which":"<name>"}
// {"verb":"list","what":"destination","which":"<id>">}
//
// {"verb":"list","what":"stream","which":"all"}
// {"verb":"list","what":"destination","which":"all"}
//
// {"verb":"delete","what":"stream","which":"<which>"}
// {"verb":"delete","what":"destination","which":"<id>">}
//
// {"verb":"delete","what":"stream","which":"all"}
// {"verb":"delete","what":"destination","which":"all"}
//
// Which is adapted from the REST-like API
//
// destination: POST {"stream":"video0","destination":"wss://<some.relay.server>/in/video0","id":"0"} /api/destinations
// stream: POST {"stream":"/stream/front/large","feeds":["video0","audio0"]} /api/streams
// GET /api/streams/all
// GET /api/destinations/all
// DELETE /api/streams</stream_name>
// DELETE /api/destinations</id>
// DELETE /api/streams/all
// DELETE /api/destinations/all

func (app *App) handleAdminMessage(msg []byte) ([]byte, error) {

	var cmd Command //map[string]*json.RawMessage
	var reply []byte

	err := json.Unmarshal(msg, &cmd)

	if err != nil {
		return reply, errBadCommand
	}
	if cmd.Verb == "healthcheck" {
		reply = []byte(`{"healthcheck":"ok"}`)
	} else {
		switch cmd.What {
		case "destination":
			switch cmd.Verb {
			case "add":
				var rule rwc.Rule
				err = json.Unmarshal(*cmd.Rule, &rule)
				if err != nil {
					log.WithField("rule", *cmd.Rule).Errorf("error adding destination because json error %s", err.Error())
					break
				}
				rule.Stream = strings.TrimPrefix(rule.Stream, "/") //to match trimming we do in handleStreamAdd
				app.Websocket.Add <- rule
				reply, err = json.Marshal(rule)
			case "delete":
				switch cmd.Which {
				case "":
					err = errBadCommand
				case "all":
					app.Websocket.Delete <- "deleteAll"
					// don't lock ourselves out!
					if app.Opts.API != "" {
						app.Websocket.Add <- rwc.Rule{Stream: "api", Destination: app.Opts.API, ID: "apiRule"}
					}
					reply = []byte(`{"deleted":"deleteAll"}`)
				default:
					if cmd.Which != "apiRule" {
						app.Websocket.Delete <- cmd.Which
						reply = []byte(`{"deleted":"` + cmd.Which + `"}`)
					} else {
						err = errNoDeleteAPIRule
					}
				}
			case "list":
				switch cmd.Which {
				case "all":
					reply, err = json.Marshal(app.Websocket.Rules)
				default:
					reply, err = json.Marshal(app.Websocket.Rules[cmd.Which])
				}
			default:
				err = errBadCommand
			}
		case "stream":
			switch cmd.Verb {
			case "add":
				var rule agg.Rule
				err = json.Unmarshal(*cmd.Rule, &rule)
				if err != nil {
					log.WithField("rule", *cmd.Rule).Errorf("error adding stream because json error %s", err.Error())
					break
				}
				rule.Stream = strings.TrimPrefix(rule.Stream, "/") //to match trimming we do in handleStreamAdd
				app.Hub.Add <- rule
				reply, err = json.Marshal(rule)
			case "delete":
				switch cmd.Which {
				case "all":
					app.Hub.Delete <- "deleteAll"
					reply = []byte(`{"deleted":"deleteAll"}`)
				default:
					app.Hub.Delete <- cmd.Which
					reply = []byte(`{"deleted":"` + cmd.Which + `"}`)
				}
			case "list":
				switch cmd.Which {
				case "":
					err = errBadCommand
				case "all":
					reply, err = json.Marshal(app.Hub.Rules)
				default:
					var feeds []byte // manage scope of err by avoiding :=
					feeds, err = json.Marshal(app.Hub.Rules[cmd.Which])
					reply = []byte(`{"feeds":` + string(feeds) + `}`)
				}
			default:
				err = errBadCommand
			}
		default:
			err = errBadCommand
		}
	}

	return reply, err

}
