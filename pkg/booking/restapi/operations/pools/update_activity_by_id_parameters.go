// Code generated by go-swagger; DO NOT EDIT.

package pools

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"io"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"

	"github.com/timdrysdale/relay/pkg/booking/models"
)

// NewUpdateActivityByIDParams creates a new UpdateActivityByIDParams object
//
// There are no default values defined in the spec.
func NewUpdateActivityByIDParams() UpdateActivityByIDParams {

	return UpdateActivityByIDParams{}
}

// UpdateActivityByIDParams contains all the bound params for the update activity by ID operation
// typically these are obtained from a http.Request
//
// swagger:parameters updateActivityByID
type UpdateActivityByIDParams struct {

	// HTTP Request Object
	HTTPRequest *http.Request `json:"-"`

	/*
	  Required: true
	  In: body
	*/
	Activity *models.Activity
	/*
	  Required: true
	  In: path
	*/
	ActivityID string
	/*
	  Required: true
	  In: path
	*/
	PoolID string
}

// BindRequest both binds and validates a request, it assumes that complex things implement a Validatable(strfmt.Registry) error interface
// for simple values it will use straight method calls.
//
// To ensure default values, the struct must have been initialized with NewUpdateActivityByIDParams() beforehand.
func (o *UpdateActivityByIDParams) BindRequest(r *http.Request, route *middleware.MatchedRoute) error {
	var res []error

	o.HTTPRequest = r

	if runtime.HasBody(r) {
		defer r.Body.Close()
		var body models.Activity
		if err := route.Consumer.Consume(r.Body, &body); err != nil {
			if err == io.EOF {
				res = append(res, errors.Required("activity", "body", ""))
			} else {
				res = append(res, errors.NewParseError("activity", "body", "", err))
			}
		} else {
			// validate body object
			if err := body.Validate(route.Formats); err != nil {
				res = append(res, err)
			}

			ctx := validate.WithOperationRequest(context.Background())
			if err := body.ContextValidate(ctx, route.Formats); err != nil {
				res = append(res, err)
			}

			if len(res) == 0 {
				o.Activity = &body
			}
		}
	} else {
		res = append(res, errors.Required("activity", "body", ""))
	}

	rActivityID, rhkActivityID, _ := route.Params.GetOK("activity_id")
	if err := o.bindActivityID(rActivityID, rhkActivityID, route.Formats); err != nil {
		res = append(res, err)
	}

	rPoolID, rhkPoolID, _ := route.Params.GetOK("pool_id")
	if err := o.bindPoolID(rPoolID, rhkPoolID, route.Formats); err != nil {
		res = append(res, err)
	}
	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// bindActivityID binds and validates parameter ActivityID from path.
func (o *UpdateActivityByIDParams) bindActivityID(rawData []string, hasKey bool, formats strfmt.Registry) error {
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: true
	// Parameter is provided by construction from the route
	o.ActivityID = raw

	return nil
}

// bindPoolID binds and validates parameter PoolID from path.
func (o *UpdateActivityByIDParams) bindPoolID(rawData []string, hasKey bool, formats strfmt.Registry) error {
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: true
	// Parameter is provided by construction from the route
	o.PoolID = raw

	return nil
}
