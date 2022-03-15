package file

import (
	"context"
	"fmt"
	"io"
	"time"

	log "github.com/sirupsen/logrus"
)

// FormatLine returns a string representing a Line, ready for writing to file
func FormatLine(line Line) string {
	return fmt.Sprintf("[%s] %s\n", line.Time.Format(time.RFC3339Nano), line.Content)
}

// Write writes the line to the file, after formatting, returning when
// the context is cancelled, or the in channel is closed.
func Write(ctx context.Context, in chan Line, w io.Writer) {

	for {
		select {
		case <-ctx.Done():
			return // avoid leaking the goro
		case line, ok := <-in:
			if !ok {
				return // avoid leaking the goro
			}
			_, err := w.Write([]byte(FormatLine(line)))
			if err != nil {
				log.Errorf("error writing to file: %s", err.Error())
			}
		}
	}
}
