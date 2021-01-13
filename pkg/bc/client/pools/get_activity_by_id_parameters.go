// Code generated by go-swagger; DO NOT EDIT.

package pools

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"context"
	"net/http"
	"time"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	cr "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
)

// NewGetActivityByIDParams creates a new GetActivityByIDParams object
// with the default values initialized.
func NewGetActivityByIDParams() *GetActivityByIDParams {
	var ()
	return &GetActivityByIDParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewGetActivityByIDParamsWithTimeout creates a new GetActivityByIDParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewGetActivityByIDParamsWithTimeout(timeout time.Duration) *GetActivityByIDParams {
	var ()
	return &GetActivityByIDParams{

		timeout: timeout,
	}
}

// NewGetActivityByIDParamsWithContext creates a new GetActivityByIDParams object
// with the default values initialized, and the ability to set a context for a request
func NewGetActivityByIDParamsWithContext(ctx context.Context) *GetActivityByIDParams {
	var ()
	return &GetActivityByIDParams{

		Context: ctx,
	}
}

// NewGetActivityByIDParamsWithHTTPClient creates a new GetActivityByIDParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewGetActivityByIDParamsWithHTTPClient(client *http.Client) *GetActivityByIDParams {
	var ()
	return &GetActivityByIDParams{
		HTTPClient: client,
	}
}

/*GetActivityByIDParams contains all the parameters to send to the API endpoint
for the get activity by ID operation typically these are written to a http.Request
*/
type GetActivityByIDParams struct {

	/*ActivityID*/
	ActivityID string
	/*Details
	  True returns all available details, false just description.

	*/
	Details *bool
	/*PoolID*/
	PoolID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the get activity by ID params
func (o *GetActivityByIDParams) WithTimeout(timeout time.Duration) *GetActivityByIDParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get activity by ID params
func (o *GetActivityByIDParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get activity by ID params
func (o *GetActivityByIDParams) WithContext(ctx context.Context) *GetActivityByIDParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get activity by ID params
func (o *GetActivityByIDParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get activity by ID params
func (o *GetActivityByIDParams) WithHTTPClient(client *http.Client) *GetActivityByIDParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get activity by ID params
func (o *GetActivityByIDParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithActivityID adds the activityID to the get activity by ID params
func (o *GetActivityByIDParams) WithActivityID(activityID string) *GetActivityByIDParams {
	o.SetActivityID(activityID)
	return o
}

// SetActivityID adds the activityId to the get activity by ID params
func (o *GetActivityByIDParams) SetActivityID(activityID string) {
	o.ActivityID = activityID
}

// WithDetails adds the details to the get activity by ID params
func (o *GetActivityByIDParams) WithDetails(details *bool) *GetActivityByIDParams {
	o.SetDetails(details)
	return o
}

// SetDetails adds the details to the get activity by ID params
func (o *GetActivityByIDParams) SetDetails(details *bool) {
	o.Details = details
}

// WithPoolID adds the poolID to the get activity by ID params
func (o *GetActivityByIDParams) WithPoolID(poolID string) *GetActivityByIDParams {
	o.SetPoolID(poolID)
	return o
}

// SetPoolID adds the poolId to the get activity by ID params
func (o *GetActivityByIDParams) SetPoolID(poolID string) {
	o.PoolID = poolID
}

// WriteToRequest writes these params to a swagger request
func (o *GetActivityByIDParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param activity_id
	if err := r.SetPathParam("activity_id", o.ActivityID); err != nil {
		return err
	}

	if o.Details != nil {

		// query param details
		var qrDetails bool
		if o.Details != nil {
			qrDetails = *o.Details
		}
		qDetails := swag.FormatBool(qrDetails)
		if qDetails != "" {
			if err := r.SetQueryParam("details", qDetails); err != nil {
				return err
			}
		}

	}

	// path param pool_id
	if err := r.SetPathParam("pool_id", o.PoolID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
