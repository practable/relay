package shellbar

import (
	"encoding/json"
	"math"
	"sync"
	"time"

	"github.com/eclesh/welford"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
)

// StatsClient starts a routine which sends stats reports on demand.
func statsClient(closed <-chan struct{}, wg *sync.WaitGroup, hub *Hub, config Config) {

	tx := &Frames{size: welford.New(), ns: welford.New(), mu: &sync.RWMutex{}}
	rx := &Frames{size: welford.New(), ns: welford.New(), mu: &sync.RWMutex{}}
	stats := &Stats{connectedAt: time.Now(), tx: tx, rx: rx}

	client := &Client{hub: hub,
		send:       make(chan message, 256),
		topic:      "stats",
		stats:      stats,
		name:       "stats-generator-" + uuid.New().String(),
		audience:   config.Audience,
		userAgent:  "shellbar",
		remoteAddr: "internal",
		canRead:    true,
		canWrite:   true,
	}
	client.hub.register <- client

	go client.statsReporter(closed, wg)

}

// StatsReporter sends a stats update in response to {"cmd":"update"}.
func (c *Client) statsReporter(closed <-chan struct{}, wg *sync.WaitGroup) {

	defer wg.Done()

	var sc StatsCommand

	for {

		// TODO consider this scheme again
		// update on receiving a message
		// or if 60 seconds since last update
		// whichever is sooner
		// newly connecting pages can send a message
		time.Sleep(time.Second) //rate limit to one message a second
		select {
		case <-closed:
			log.Trace("StatsReporter closed")
			return
		case msg, ok := <-c.send: // received a message from hub

			if !ok {
				return //send is closed, so we are finished
			}

			err := json.Unmarshal(msg.data, &sc)

			if err != nil {
				log.WithFields(log.Fields{"error": err, "msg": string(msg.data)}).Trace("statsReporter could not marshall into json")
			}

			log.WithField("cmd", sc.Command).Trace("statsReporter received command")

			doUpdate := false

			if sc.Command == "update" {
				doUpdate = true
			}

			n := len(c.send)
			for i := 0; i < n; i++ {
				msg, ok = <-c.send
				if !ok {
					return //send is closed, so we are finished
				}

				err = json.Unmarshal(msg.data, &sc)

				if err != nil {
					log.WithFields(log.Fields{"error": err, "msg": string(msg.data)}).Trace("statsReporter could not marshall into json")
				}

				log.WithField("cmd", sc.Command).Trace("statsReporter received command")

				if sc.Command == "update" {
					doUpdate = true
				}
			}

			log.WithField("doUpdate", doUpdate).Trace("statsReporter do update?")

			if !doUpdate { //don't send updated stats
				continue
			}

		case <-time.After(60 * time.Second):
			log.Trace("StatsReporter routine send...")
		}

		var reports []*ClientReport

		c.hub.mu.RLock()
		for _, topic := range c.hub.clients {
			for client := range topic {

				client.stats.tx.mu.RLock()

				var tx ReportStats

				if client.stats.tx.size.Count() > 0 {
					tx = ReportStats{
						Last: time.Since(client.stats.tx.last).String(),
						Size: math.Round(client.stats.tx.size.Mean()),
						Fps:  fpsFromNs(client.stats.tx.ns.Mean()),
					}
				} else {
					tx = ReportStats{
						Last: "Never",
						Size: 0,
						Fps:  0,
					}
				}

				client.stats.tx.mu.RUnlock()

				client.stats.rx.mu.RLock()
				var rx ReportStats

				if client.stats.rx.size.Count() > 0 {
					rx = ReportStats{
						Last: time.Since(client.stats.rx.last).String(),
						Size: math.Round(client.stats.rx.size.Mean()),
						Fps:  fpsFromNs(client.stats.rx.ns.Mean()),
					}
				} else {
					rx = ReportStats{
						Last: "Never",
						Size: 0,
						Fps:  0,
					}
				}

				client.stats.rx.mu.RUnlock()

				report := &ClientReport{
					Topic:      client.topic,
					CanRead:    client.canRead,
					CanWrite:   client.canWrite,
					Connected:  client.stats.connectedAt.String(),
					RemoteAddr: client.remoteAddr,
					UserAgent:  client.userAgent,
					Stats: RxTx{
						Tx: tx,
						Rx: rx,
					},
				}

				reports = append(reports, report)

			} //for client in topic
		} // for topic in hub
		c.hub.mu.RUnlock()
		reportsData, err := json.Marshal(reports)
		if err != nil {
			log.WithField("error", err).Error("statsReporter marshalling JSON")
			return
		}
		// broadcast stats back to the hub (i.e. and anyone listening to this topic)
		c.hub.broadcast <- message{sender: *c, data: reportsData, mt: websocket.TextMessage}

	}
}
