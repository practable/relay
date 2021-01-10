package booking

import (
	"github.com/go-openapi/runtime/middleware"
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
		return middleware.NotImplemented("operation admin.GetStoreStatus has not yet been implemented")
	}
}

func importPoolStore(ps *pool.PoolStore, l *bookingstore.Limit) func(admin.ImportPoolStoreParams, interface{}) middleware.Responder {

	return func(params admin.ImportPoolStoreParams, principal interface{}) middleware.Responder {
		return middleware.NotImplemented("operation admin.ImportPoolStore has not yet been implemented")
	}
}
