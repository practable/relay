/*
   client is a public wrapper for internal/reconws so that
   we can expose this useful code but without having to
   commit to publically declaring the specifics of the
   internal API, as this may change later.
*/

package status

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/practable/relay/pkg/client"
	log "github.com/sirupsen/logrus"
)

// Report represents information about a client's connection, permissions, and statistics
// Used to unmarshal report data in the format from internal/crossbar/models.ClientReport

type Report struct {
	Topic      string    `json:"topic"`
	CanRead    bool      `json:"canRead"`
	CanWrite   bool      `json:"canWrite"`
	Connected  time.Time `json:"connected"`
	ExpiresAt  time.Time `json:"expiresAt"`
	RemoteAddr string    `json:"remoteAddr"`
	UserAgent  string    `json:"userAgent"`
	Stats      RxTx      `json:"stats"`
}

//RxTx represents statistics for both receive and transmit
type RxTx struct {
	Tx Statistics `json:"tx"`
	Rx Statistics `json:"rx"`
}

// Statistics represents statistics about what has been sent/received
type Statistics struct {
	Last  time.Duration `json:"last"` //duration since last message
	Size  float64       `json:"size"`
	FPS   float64       `json:"fps"`
	Never bool          `json:"never"`
}

type Status struct {
	c      *client.Client
	Status chan []Report
}

func New() *Status {
	return &Status{
		c:      client.New(),
		Status: make(chan []Report),
	}
}

func (s *Status) Connect(ctx context.Context, to, token string) {
	go func() {
	LOOP:
		for {
			select {
			case <-ctx.Done():
				log.Trace("status receive stopped permanently")
				break LOOP //we're done
			case msg := <-s.c.Receive:
				log.WithField("content", string(msg.Content)).Trace("status received")
				var reports []Report
				err := json.Unmarshal(msg.Content, &reports)
				if err != nil {
					log.WithFields(log.Fields{"error": err.Error(), "content": string(msg.Content)}).Error("could not unmarshal status report")
					continue
				}
				s.Status <- reports
			}
		}
	}()
	s.c.Connect(ctx, to, token)
}

func (s *Statistics) UnmarshalJSON(data []byte) (err error) {

	var tmp struct {
		// durations are set to string for now
		Last string  `json:"last"` //duration since last message
		Size float64 `json:"size"`
		FPS  float64 `json:"fps"`
		// There is no Never field in the reports
	}

	if err = json.Unmarshal(data, &tmp); err != nil {
		return err
	}

	tmp.Last = strings.TrimSpace(strings.ToLower(tmp.Last))

	// set default durations
	if tmp.Last == "never" || tmp.Last == "" {
		s.Never = true
		tmp.Last = "999h"
	} else {
		s.Never = false
	}

	// parse durations

	ld, err := time.ParseDuration(tmp.Last)
	if err != nil {
		return err
	}

	s.Last = ld
	s.Size = tmp.Size
	s.FPS = tmp.FPS

	return nil

}
