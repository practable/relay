// Code generated by go-swagger; DO NOT EDIT.

package admin

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// ImportPoolStoreHandlerFunc turns a function with the right signature into a import pool store handler
type ImportPoolStoreHandlerFunc func(ImportPoolStoreParams, interface{}) middleware.Responder

// Handle executing the request and returning a response
func (fn ImportPoolStoreHandlerFunc) Handle(params ImportPoolStoreParams, principal interface{}) middleware.Responder {
	return fn(params, principal)
}

// ImportPoolStoreHandler interface for that can handle valid import pool store params
type ImportPoolStoreHandler interface {
	Handle(ImportPoolStoreParams, interface{}) middleware.Responder
}

// NewImportPoolStore creates a new http.Handler for the import pool store operation
func NewImportPoolStore(ctx *middleware.Context, handler ImportPoolStoreHandler) *ImportPoolStore {
	return &ImportPoolStore{Context: ctx, Handler: handler}
}

/* ImportPoolStore swagger:route POST /admin/poolstore admin importPoolStore

Import new current state

Import a new pool store including bookings

*/
type ImportPoolStore struct {
	Context *middleware.Context
	Handler ImportPoolStoreHandler
}

func (o *ImportPoolStore) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		r = rCtx
	}
	var Params = NewImportPoolStoreParams()
	uprinc, aCtx, err := o.Context.Authorize(r, route)
	if err != nil {
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}
	if aCtx != nil {
		r = aCtx
	}
	var principal interface{}
	if uprinc != nil {
		principal = uprinc.(interface{}) // this is really a interface{}, I promise
	}

	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params, principal) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}
