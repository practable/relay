package book

import (
	"context"

	"github.com/timdrysdale/relay/pkg/booking"
	"github.com/timdrysdale/relay/pkg/pool"
)

func Book(ctx context.Context, port int, bookingDuration int64, host, secret string) {

	ps := pool.NewPoolStore().
		WithSecret(secret).
		WithBookingTokenDuration(bookingDuration)

	go booking.API(ctx, port, host, secret, ps)

	<-ctx.Done()
}
