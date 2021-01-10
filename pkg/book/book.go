package book

import (
	"context"
	"time"

	"github.com/timdrysdale/relay/pkg/booking"
	"github.com/timdrysdale/relay/pkg/bookingstore"
	"github.com/timdrysdale/relay/pkg/pool"
)

func Book(ctx context.Context, port int, bookingDuration int64, host, secret string) {

	ps := pool.NewPoolStore().
		WithSecret(secret).
		WithBookingTokenDuration(bookingDuration)

	l := bookingstore.New(ctx).WithFlush(time.Hour).WithMax(2).WithProvisionalPeriod(time.Minute)

	go booking.API(ctx, port, host, secret, ps, l)

	<-ctx.Done()
}
