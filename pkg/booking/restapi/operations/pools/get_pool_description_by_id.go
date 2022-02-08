// Code generated by go-swagger; DO NOT EDIT.

package pools

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// GetPoolDescriptionByIDHandlerFunc turns a function with the right signature into a get pool description by ID handler
type GetPoolDescriptionByIDHandlerFunc func(GetPoolDescriptionByIDParams, interface{}) middleware.Responder

// Handle executing the request and returning a response
func (fn GetPoolDescriptionByIDHandlerFunc) Handle(params GetPoolDescriptionByIDParams, principal interface{}) middleware.Responder {
	return fn(params, principal)
}

// GetPoolDescriptionByIDHandler interface for that can handle valid get pool description by ID params
type GetPoolDescriptionByIDHandler interface {
	Handle(GetPoolDescriptionByIDParams, interface{}) middleware.Responder
}

// NewGetPoolDescriptionByID creates a new http.Handler for the get pool description by ID operation
func NewGetPoolDescriptionByID(ctx *middleware.Context, handler GetPoolDescriptionByIDHandler) *GetPoolDescriptionByID {
	return &GetPoolDescriptionByID{Context: ctx, Handler: handler}
}

/* GetPoolDescriptionByID swagger:route GET /pools/{pool_id} pools getPoolDescriptionById

pools

Gets a description of the pool

*/
type GetPoolDescriptionByID struct {
	Context *middleware.Context
	Handler GetPoolDescriptionByIDHandler
}

func (o *GetPoolDescriptionByID) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		r = rCtx
	}
	var Params = NewGetPoolDescriptionByIDParams()
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
