// Code generated by go-swagger; DO NOT EDIT.

package pool

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
)

// NewPoolParams creates a new PoolParams object
// no default values defined in spec.
func NewPoolParams() PoolParams {

	return PoolParams{}
}

// PoolParams contains all the bound params for the pool operation
// typically these are obtained from a http.Request
//
// swagger:parameters pool
type PoolParams struct {

	// HTTP Request Object
	HTTPRequest *http.Request `json:"-"`

	/*Search by category
	  In: query
	*/
	Category *string
	/*Search by name
	  In: query
	*/
	Name *string
}

// BindRequest both binds and validates a request, it assumes that complex things implement a Validatable(strfmt.Registry) error interface
// for simple values it will use straight method calls.
//
// To ensure default values, the struct must have been initialized with NewPoolParams() beforehand.
func (o *PoolParams) BindRequest(r *http.Request, route *middleware.MatchedRoute) error {
	var res []error

	o.HTTPRequest = r

	qs := runtime.Values(r.URL.Query())

	qCategory, qhkCategory, _ := qs.GetOK("category")
	if err := o.bindCategory(qCategory, qhkCategory, route.Formats); err != nil {
		res = append(res, err)
	}

	qName, qhkName, _ := qs.GetOK("name")
	if err := o.bindName(qName, qhkName, route.Formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// bindCategory binds and validates parameter Category from query.
func (o *PoolParams) bindCategory(rawData []string, hasKey bool, formats strfmt.Registry) error {
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: false
	// AllowEmptyValue: false
	if raw == "" { // empty values pass all other validations
		return nil
	}

	o.Category = &raw

	return nil
}

// bindName binds and validates parameter Name from query.
func (o *PoolParams) bindName(rawData []string, hasKey bool, formats strfmt.Registry) error {
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: false
	// AllowEmptyValue: false
	if raw == "" { // empty values pass all other validations
		return nil
	}

	o.Name = &raw

	return nil
}