package booking

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/timdrysdale/relay/pkg/booking/models"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/admin"
	"github.com/timdrysdale/relay/pkg/bookingstore"
	"github.com/timdrysdale/relay/pkg/pool"
)

func exportPoolStore(ps *pool.PoolStore, l *bookingstore.Limit) func(admin.ExportPoolStoreParams, interface{}) middleware.Responder {

	return func(params admin.ExportPoolStoreParams, principal interface{}) middleware.Responder {
		return middleware.NotImplemented("operation admin.ExportPoolStore has not yet been implemented")
	}
}

func getStoreStatus(ps *pool.PoolStore, l *bookingstore.Limit) func(admin.GetStoreStatusParams, interface{}) middleware.Responder {

	return func(params admin.GetStoreStatusParams, principal interface{}) middleware.Responder {
		//need admin status!

		_, err := isBookingAdmin(principal)

		if err != nil {
			return admin.NewGetStoreStatusUnauthorized().WithPayload(err.Error())
		}

		status := &models.StoreStatus{
			Activities:      int64(l.GetAllActivitiesCount()),
			Bookings:        int64(l.GetAllSessionCount()),
			Groups:          int64(ps.GetAllGroupsCount()),
			Pools:           int64(ps.GetAllPoolCount()),
			LastBookingEnds: float64(l.GetLastBookingEnds()),
			Locked:          l.GetLockBookings(),
		}
		return admin.NewGetStoreStatusOK().WithPayload(status)
	}
}

func importPoolStore(ps *pool.PoolStore, l *bookingstore.Limit) func(admin.ImportPoolStoreParams, interface{}) middleware.Responder {

	return func(params admin.ImportPoolStoreParams, principal interface{}) middleware.Responder {
		return middleware.NotImplemented("operation admin.ImportPoolStore has not yet been implemented")
	}
}
