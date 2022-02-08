// Code generated by go-swagger; DO NOT EDIT.

package groups

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
)

// NewGetPoolsByGroupIDParams creates a new GetPoolsByGroupIDParams object
//
// There are no default values defined in the spec.
func NewGetPoolsByGroupIDParams() GetPoolsByGroupIDParams {

	return GetPoolsByGroupIDParams{}
}

// GetPoolsByGroupIDParams contains all the bound params for the get pools by group ID operation
// typically these are obtained from a http.Request
//
// swagger:parameters getPoolsByGroupID
type GetPoolsByGroupIDParams struct {

	// HTTP Request Object
	HTTPRequest *http.Request `json:"-"`

	/*
	  Required: true
	  In: path
	*/
	GroupID string
}

// BindRequest both binds and validates a request, it assumes that complex things implement a Validatable(strfmt.Registry) error interface
// for simple values it will use straight method calls.
//
// To ensure default values, the struct must have been initialized with NewGetPoolsByGroupIDParams() beforehand.
func (o *GetPoolsByGroupIDParams) BindRequest(r *http.Request, route *middleware.MatchedRoute) error {
	var res []error

	o.HTTPRequest = r

	rGroupID, rhkGroupID, _ := route.Params.GetOK("group_id")
	if err := o.bindGroupID(rGroupID, rhkGroupID, route.Formats); err != nil {
		res = append(res, err)
	}
	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// bindGroupID binds and validates parameter GroupID from path.
func (o *GetPoolsByGroupIDParams) bindGroupID(rawData []string, hasKey bool, formats strfmt.Registry) error {
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: true
	// Parameter is provided by construction from the route
	o.GroupID = raw

	return nil
}
