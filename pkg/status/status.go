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
	"fmt"
	"strings"
	"time"

	"github.com/practable/relay/pkg/client"
	log "github.com/sirupsen/logrus"
)

// Report represents information about a client's connection, permissions, and statistics
// Used to unmarshal report data in the format from internal/crossbar/models.ClientReport

type Report struct {
	CanRead    bool      `json:"canRead"`
	CanWrite   bool      `json:"canWrite"`
	Connected  time.Time `json:"connected"`
	ExpiresAt  time.Time `json:"expiresAt"`
	RemoteAddr string    `json:"remoteAddr"`
	Scopes     []string  `json:"scopes"`
	Stats      RxTx      `json:"stats"`
	Topic      string    `json:"topic"`
	UserAgent  string    `json:"userAgent"`
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

	var tmpString struct {
		// durations are set to string for now
		Last string  `json:"last"` //duration since last message
		Size float64 `json:"size"`
		FPS  float64 `json:"fps"`
		// There is no Never field in the reports
	}

	// added to support marshalling of experimental reports into json/yaml in status,
	// which results in durations in nanoseconds
	var tmpNumber struct {
		// durations are set to string for now
		Last int64   `json:"last"` //duration since last message
		Size float64 `json:"size"`
		FPS  float64 `json:"fps"`
		// There is no Never field in the reports
	}

	errString := json.Unmarshal(data, &tmpString)

	if errString == nil {

		tmpString.Last = strings.TrimSpace(strings.ToLower(tmpString.Last))

		// set default durations
		if tmpString.Last == "never" || tmpString.Last == "" {
			s.Never = true
			tmpString.Last = "999h"
		} else {
			s.Never = false
		}

		// parse durations

		ld, err := time.ParseDuration(tmpString.Last)
		if err != nil {
			return err
		}

		s.Last = ld
		s.Size = tmpString.Size
		s.FPS = tmpString.FPS

		return nil
	}

	errNumber := json.Unmarshal(data, &tmpNumber)

	if errNumber == nil {

		// duration is in nanoseconds
		s.Last = time.Duration(tmpNumber.Last) * time.Nanosecond
		s.Size = tmpNumber.Size
		s.FPS = tmpNumber.FPS

		return nil
	}

	return fmt.Errorf("%s %s", errString.Error(), errNumber.Error())

}
