package booking

import (
	b64 "encoding/base64"

	"github.com/go-openapi/runtime/middleware"
	"github.com/timdrysdale/relay/pkg/booking/models"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/admin"
	"github.com/timdrysdale/relay/pkg/bookingstore"
	"github.com/timdrysdale/relay/pkg/pool"
)

func exportPoolStore(ps *pool.PoolStore, l *bookingstore.Limit) func(admin.ExportPoolStoreParams, interface{}) middleware.Responder {

	return func(params admin.ExportPoolStoreParams, principal interface{}) middleware.Responder {

		_, err := isBookingAdmin(principal)

		if err != nil {
			return admin.NewExportPoolStoreUnauthorized().WithPayload(err.Error())
		}

		booking, err := l.ExportAll()

		if err != nil {
			return admin.NewExportPoolStoreInternalServerError().WithPayload(err.Error())
		}
		bookingEnc := b64.StdEncoding.EncodeToString(booking)
		pool, err := ps.ExportAll()

		if err != nil {
			return admin.NewExportPoolStoreInternalServerError().WithPayload(err.Error())
		}

		poolEnc := b64.StdEncoding.EncodeToString(pool)

		store := &models.Poolstore{
			Booking: &bookingEnc,
			Pool:    &poolEnc,
		}

		return admin.NewExportPoolStoreOK().WithPayload(store)
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

		_, err := isBookingAdmin(principal)

		if err != nil {
			return admin.NewImportPoolStoreUnauthorized().WithPayload(err.Error())
		}

		if params.Poolstore == nil {
			return admin.NewImportPoolStoreInternalServerError().WithPayload("Empty poolstore in request")
		}
		if params.Poolstore.Booking == nil {
			return admin.NewImportPoolStoreInternalServerError().WithPayload("Empty booking in request")
		}

		if params.Poolstore.Pool == nil {
			return admin.NewImportPoolStoreInternalServerError().WithPayload("Empty pool in request")
		}

		bookingDec, err := b64.StdEncoding.DecodeString(*params.Poolstore.Booking)

		if err != nil {
			return admin.NewImportPoolStoreInternalServerError().WithPayload("booking:" + err.Error())
		}

		poolDec, err := b64.StdEncoding.DecodeString(*params.Poolstore.Pool)

		if err != nil {
			return admin.NewImportPoolStoreInternalServerError().WithPayload("pool:" + err.Error())
		}

		newl, err := bookingstore.ImportAll(l, bookingDec)

		if err != nil {
			return admin.NewImportPoolStoreInternalServerError().WithPayload("import booking:" + err.Error())
		}

		bookingstore.CopyStore(newl, l)

		newps, err := pool.ImportAll(poolDec)

		if err != nil {
			return admin.NewImportPoolStoreInternalServerError().WithPayload("import pool:" + err.Error())
		}

		// can't do ps = newps because the ps we have here is local scope
		// and outside this function we won't see the change
		pool.CopyStore(newps, ps)

		status := &models.StoreStatus{
			Activities:      int64(l.GetAllActivitiesCount()),
			Bookings:        int64(l.GetAllSessionCount()),
			Groups:          int64(ps.GetAllGroupsCount()),
			Pools:           int64(ps.GetAllPoolCount()),
			LastBookingEnds: float64(l.GetLastBookingEnds()),
			Locked:          l.GetLockBookings(),
		}

		return admin.NewImportPoolStoreOK().WithPayload(status)
	}
}

func setLock(ps *pool.PoolStore, l *bookingstore.Limit) func(admin.SetLockParams, interface{}) middleware.Responder {
	return func(params admin.SetLockParams, principal interface{}) middleware.Responder {
		_, err := isBookingAdmin(principal)

		if err != nil {
			return admin.NewSetLockUnauthorized().WithPayload(err.Error())
		}

		if params.Lock == true {
			l.LockBookings()
		} else {
			l.UnlockBookings()
		}
		status := &models.StoreStatus{
			Activities:      int64(l.GetAllActivitiesCount()),
			Bookings:        int64(l.GetAllSessionCount()),
			Groups:          int64(ps.GetAllGroupsCount()),
			Pools:           int64(ps.GetAllPoolCount()),
			LastBookingEnds: float64(l.GetLastBookingEnds()),
			Locked:          l.GetLockBookings(),
		}
		return admin.NewSetLockOK().WithPayload(status)
	}
}
