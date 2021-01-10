// This file is safe to edit. Once it exists it will not be overwritten

package restapi

import (
	"crypto/tls"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"

	"github.com/timdrysdale/relay/pkg/booking/restapi/operations"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/admin"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/groups"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/login"
	"github.com/timdrysdale/relay/pkg/booking/restapi/operations/pools"
)

//go:generate swagger generate server --target ../../booking --name Booking --spec ../../../api/openapi-spec/booking.yml --principal interface{} --exclude-main

func configureFlags(api *operations.BookingAPI) {
	// api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{ ... }
}

func configureAPI(api *operations.BookingAPI) http.Handler {
	// configure the api here
	api.ServeError = errors.ServeError

	// Set your custom logger if needed. Default one is log.Printf
	// Expected interface func(string, ...interface{})
	//
	// Example:
	// api.Logger = log.Printf

	api.UseSwaggerUI()
	// To continue using redoc as your UI, uncomment the following line
	// api.UseRedoc()

	api.JSONConsumer = runtime.JSONConsumer()

	api.JSONProducer = runtime.JSONProducer()

	// Applies when the "Authorization" header is set
	if api.BearerAuth == nil {
		api.BearerAuth = func(token string) (interface{}, error) {
			return nil, errors.NotImplemented("api key auth (Bearer) Authorization from header param [Authorization] has not yet been implemented")
		}
	}

	// Set your custom authorizer if needed. Default one is security.Authorized()
	// Expected interface runtime.Authorizer
	//
	// Example:
	// api.APIAuthorizer = security.Authorized()
	if api.PoolsAddActivityByPoolIDHandler == nil {
		api.PoolsAddActivityByPoolIDHandler = pools.AddActivityByPoolIDHandlerFunc(func(params pools.AddActivityByPoolIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation pools.AddActivityByPoolID has not yet been implemented")
		})
	}
	if api.GroupsAddNewGroupHandler == nil {
		api.GroupsAddNewGroupHandler = groups.AddNewGroupHandlerFunc(func(params groups.AddNewGroupParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation groups.AddNewGroup has not yet been implemented")
		})
	}
	if api.PoolsAddNewPoolHandler == nil {
		api.PoolsAddNewPoolHandler = pools.AddNewPoolHandlerFunc(func(params pools.AddNewPoolParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation pools.AddNewPool has not yet been implemented")
		})
	}
	if api.GroupsAddPoolsByGroupIDHandler == nil {
		api.GroupsAddPoolsByGroupIDHandler = groups.AddPoolsByGroupIDHandlerFunc(func(params groups.AddPoolsByGroupIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation groups.AddPoolsByGroupID has not yet been implemented")
		})
	}
	if api.PoolsDeleteActivityByIDHandler == nil {
		api.PoolsDeleteActivityByIDHandler = pools.DeleteActivityByIDHandlerFunc(func(params pools.DeleteActivityByIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation pools.DeleteActivityByID has not yet been implemented")
		})
	}
	if api.PoolsDeletePoolHandler == nil {
		api.PoolsDeletePoolHandler = pools.DeletePoolHandlerFunc(func(params pools.DeletePoolParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation pools.DeletePool has not yet been implemented")
		})
	}
	if api.GroupsDeletePoolsByGroupIDHandler == nil {
		api.GroupsDeletePoolsByGroupIDHandler = groups.DeletePoolsByGroupIDHandlerFunc(func(params groups.DeletePoolsByGroupIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation groups.DeletePoolsByGroupID has not yet been implemented")
		})
	}
	if api.AdminExportPoolStoreHandler == nil {
		api.AdminExportPoolStoreHandler = admin.ExportPoolStoreHandlerFunc(func(params admin.ExportPoolStoreParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation admin.ExportPoolStore has not yet been implemented")
		})
	}
	if api.PoolsGetActivityByIDHandler == nil {
		api.PoolsGetActivityByIDHandler = pools.GetActivityByIDHandlerFunc(func(params pools.GetActivityByIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation pools.GetActivityByID has not yet been implemented")
		})
	}
	if api.PoolsGetAllPoolsHandler == nil {
		api.PoolsGetAllPoolsHandler = pools.GetAllPoolsHandlerFunc(func(params pools.GetAllPoolsParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation pools.GetAllPools has not yet been implemented")
		})
	}
	if api.GroupsGetGroupDescriptionByIDHandler == nil {
		api.GroupsGetGroupDescriptionByIDHandler = groups.GetGroupDescriptionByIDHandlerFunc(func(params groups.GetGroupDescriptionByIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation groups.GetGroupDescriptionByID has not yet been implemented")
		})
	}
	if api.GroupsGetGroupIDByNameHandler == nil {
		api.GroupsGetGroupIDByNameHandler = groups.GetGroupIDByNameHandlerFunc(func(params groups.GetGroupIDByNameParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation groups.GetGroupIDByName has not yet been implemented")
		})
	}
	if api.PoolsGetPoolDescriptionByIDHandler == nil {
		api.PoolsGetPoolDescriptionByIDHandler = pools.GetPoolDescriptionByIDHandlerFunc(func(params pools.GetPoolDescriptionByIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation pools.GetPoolDescriptionByID has not yet been implemented")
		})
	}
	if api.PoolsGetPoolStatusByIDHandler == nil {
		api.PoolsGetPoolStatusByIDHandler = pools.GetPoolStatusByIDHandlerFunc(func(params pools.GetPoolStatusByIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation pools.GetPoolStatusByID has not yet been implemented")
		})
	}
	if api.GroupsGetPoolsByGroupIDHandler == nil {
		api.GroupsGetPoolsByGroupIDHandler = groups.GetPoolsByGroupIDHandlerFunc(func(params groups.GetPoolsByGroupIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation groups.GetPoolsByGroupID has not yet been implemented")
		})
	}
	if api.AdminGetStoreStatusHandler == nil {
		api.AdminGetStoreStatusHandler = admin.GetStoreStatusHandlerFunc(func(params admin.GetStoreStatusParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation admin.GetStoreStatus has not yet been implemented")
		})
	}
	if api.AdminImportPoolStoreHandler == nil {
		api.AdminImportPoolStoreHandler = admin.ImportPoolStoreHandlerFunc(func(params admin.ImportPoolStoreParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation admin.ImportPoolStore has not yet been implemented")
		})
	}
	if api.LoginLoginHandler == nil {
		api.LoginLoginHandler = login.LoginHandlerFunc(func(params login.LoginParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation login.Login has not yet been implemented")
		})
	}
	if api.GroupsReplacePoolsByGroupIDHandler == nil {
		api.GroupsReplacePoolsByGroupIDHandler = groups.ReplacePoolsByGroupIDHandlerFunc(func(params groups.ReplacePoolsByGroupIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation groups.ReplacePoolsByGroupID has not yet been implemented")
		})
	}
	if api.PoolsRequestSessionByPoolIDHandler == nil {
		api.PoolsRequestSessionByPoolIDHandler = pools.RequestSessionByPoolIDHandlerFunc(func(params pools.RequestSessionByPoolIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation pools.RequestSessionByPoolID has not yet been implemented")
		})
	}
	if api.AdminSetLockHandler == nil {
		api.AdminSetLockHandler = admin.SetLockHandlerFunc(func(params admin.SetLockParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation admin.SetLock has not yet been implemented")
		})
	}
	if api.PoolsUpdateActivityByIDHandler == nil {
		api.PoolsUpdateActivityByIDHandler = pools.UpdateActivityByIDHandlerFunc(func(params pools.UpdateActivityByIDParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation pools.UpdateActivityByID has not yet been implemented")
		})
	}

	api.PreServerShutdown = func() {}

	api.ServerShutdown = func() {}

	return setupGlobalMiddleware(api.Serve(setupMiddlewares))
}

// The TLS configuration before HTTPS server starts.
func configureTLS(tlsConfig *tls.Config) {
	// Make all necessary changes to the TLS configuration here.
}

// As soon as server is initialized but not run yet, this function will be called.
// If you need to modify a config, store server instance to stop it individually later, this is the place.
// This function can be called multiple times, depending on the number of serving schemes.
// scheme value will be set accordingly: "http", "https" or "unix"
func configureServer(s *http.Server, scheme, addr string) {
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation
func setupMiddlewares(handler http.Handler) http.Handler {
	return handler
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return handler
}
