// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// NewAllowParams creates a new AllowParams object
//
// There are no default values defined in the spec.
func NewAllowParams() AllowParams {

	return AllowParams{}
}

// AllowParams contains all the bound params for the allow operation
// typically these are obtained from a http.Request
//
// swagger:parameters allow
type AllowParams struct {

	// HTTP Request Object
	HTTPRequest *http.Request `json:"-"`

	/*
	  Required: true
	  In: query
	*/
	Bid string
	/*
	  Required: true
	  In: query
	*/
	Exp int64
}

// BindRequest both binds and validates a request, it assumes that complex things implement a Validatable(strfmt.Registry) error interface
// for simple values it will use straight method calls.
//
// To ensure default values, the struct must have been initialized with NewAllowParams() beforehand.
func (o *AllowParams) BindRequest(r *http.Request, route *middleware.MatchedRoute) error {
	var res []error

	o.HTTPRequest = r

	qs := runtime.Values(r.URL.Query())

	qBid, qhkBid, _ := qs.GetOK("bid")
	if err := o.bindBid(qBid, qhkBid, route.Formats); err != nil {
		res = append(res, err)
	}

	qExp, qhkExp, _ := qs.GetOK("exp")
	if err := o.bindExp(qExp, qhkExp, route.Formats); err != nil {
		res = append(res, err)
	}
	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// bindBid binds and validates parameter Bid from query.
func (o *AllowParams) bindBid(rawData []string, hasKey bool, formats strfmt.Registry) error {
	if !hasKey {
		return errors.Required("bid", "query", rawData)
	}
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: true
	// AllowEmptyValue: false

	if err := validate.RequiredString("bid", "query", raw); err != nil {
		return err
	}
	o.Bid = raw

	return nil
}

// bindExp binds and validates parameter Exp from query.
func (o *AllowParams) bindExp(rawData []string, hasKey bool, formats strfmt.Registry) error {
	if !hasKey {
		return errors.Required("exp", "query", rawData)
	}
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: true
	// AllowEmptyValue: false

	if err := validate.RequiredString("exp", "query", raw); err != nil {
		return err
	}

	value, err := swag.ConvertInt64(raw)
	if err != nil {
		return errors.InvalidType("exp", "query", "int64", raw)
	}
	o.Exp = value

	return nil
}
