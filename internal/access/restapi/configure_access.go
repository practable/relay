// This file is safe to edit. Once it exists it will not be overwritten

package restapi

import (
	"crypto/tls"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"

	"github.com/practable/relay/internal/access/restapi/operations"
)

//go:generate swagger generate server --target ../../access --name Access --spec ../../../api/access.yml --principal interface{} --exclude-main

func configureFlags(api *operations.AccessAPI) {
	// api.CommandLineOptionsGroups = []swag.CommandLineOptionsGroup{ ... }
}

func configureAPI(api *operations.AccessAPI) http.Handler {
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

	if api.AllowHandler == nil {
		api.AllowHandler = operations.AllowHandlerFunc(func(params operations.AllowParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation operations.Allow has not yet been implemented")
		})
	}
	if api.DenyHandler == nil {
		api.DenyHandler = operations.DenyHandlerFunc(func(params operations.DenyParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation operations.Deny has not yet been implemented")
		})
	}
	if api.GetStatusHandler == nil {
		api.GetStatusHandler = operations.GetStatusHandlerFunc(func(params operations.GetStatusParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation operations.GetStatus has not yet been implemented")
		})
	}
	if api.ListAllowedHandler == nil {
		api.ListAllowedHandler = operations.ListAllowedHandlerFunc(func(params operations.ListAllowedParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation operations.ListAllowed has not yet been implemented")
		})
	}
	if api.ListDeniedHandler == nil {
		api.ListDeniedHandler = operations.ListDeniedHandlerFunc(func(params operations.ListDeniedParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation operations.ListDenied has not yet been implemented")
		})
	}
	if api.SessionHandler == nil {
		api.SessionHandler = operations.SessionHandlerFunc(func(params operations.SessionParams, principal interface{}) middleware.Responder {
			return middleware.NotImplemented("operation operations.Session has not yet been implemented")
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
// scheme value will be set accordingly: "http", "https" or "unix".
func configureServer(s *http.Server, scheme, addr string) {
}

// The middleware configuration is for the handler executors. These do not apply to the swagger.json document.
// The middleware executes after routing but before authentication, binding and validation.
func setupMiddlewares(handler http.Handler) http.Handler {
	return handler
}

// The middleware configuration happens before anything, this middleware also applies to serving the swagger.json document.
// So this is a good place to plug in a panic handling middleware, logging and metrics.
func setupGlobalMiddleware(handler http.Handler) http.Handler {
	return handler
}
