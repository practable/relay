package book

import (
	"context"
	"time"

	"github.com/timdrysdale/relay/pkg/booking"
	"github.com/timdrysdale/relay/pkg/bookingstore"
	"github.com/timdrysdale/relay/pkg/pool"
)

// Book creates a new bookingstore server listening at host:port,
// It accepts login tokens signed with secret, and returns a token
// that can be used to make up to 2 bookings at any time within the
// bookingDuration seconds.
func Book(ctx context.Context, port int, bookingDuration int64, host, secret string) {

	ps := pool.NewPoolStore().
		WithSecret(secret).
		WithBookingTokenDuration(bookingDuration)

	l := bookingstore.New(ctx).WithFlush(time.Hour).WithMax(2).WithProvisionalPeriod(time.Minute)

	go booking.API(ctx, port, host, secret, ps, l)

	<-ctx.Done()
}
