package file

import (
	"context"
	"fmt"
	"os"
)

// Run connects to the session and handles writing to/from files
func Run(ctx context.Context, hup chan os.Signal, session, token, logfilename, playfilename string) {

	go func() {
		for {
			select {
			case <-ctx.Done():
				return //avoid leaking this goroutine if we are cancelled
			case <-hup:
				fmt.Printf("SIGHUP detected, reopening LOG file %s\n", logfilename)
			}
		}
	}()

}
