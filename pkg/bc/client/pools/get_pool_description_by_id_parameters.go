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
)

// NewGetPoolDescriptionByIDParams creates a new GetPoolDescriptionByIDParams object
// with the default values initialized.
func NewGetPoolDescriptionByIDParams() *GetPoolDescriptionByIDParams {
	var ()
	return &GetPoolDescriptionByIDParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewGetPoolDescriptionByIDParamsWithTimeout creates a new GetPoolDescriptionByIDParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewGetPoolDescriptionByIDParamsWithTimeout(timeout time.Duration) *GetPoolDescriptionByIDParams {
	var ()
	return &GetPoolDescriptionByIDParams{

		timeout: timeout,
	}
}

// NewGetPoolDescriptionByIDParamsWithContext creates a new GetPoolDescriptionByIDParams object
// with the default values initialized, and the ability to set a context for a request
func NewGetPoolDescriptionByIDParamsWithContext(ctx context.Context) *GetPoolDescriptionByIDParams {
	var ()
	return &GetPoolDescriptionByIDParams{

		Context: ctx,
	}
}

// NewGetPoolDescriptionByIDParamsWithHTTPClient creates a new GetPoolDescriptionByIDParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewGetPoolDescriptionByIDParamsWithHTTPClient(client *http.Client) *GetPoolDescriptionByIDParams {
	var ()
	return &GetPoolDescriptionByIDParams{
		HTTPClient: client,
	}
}

/*GetPoolDescriptionByIDParams contains all the parameters to send to the API endpoint
for the get pool description by ID operation typically these are written to a http.Request
*/
type GetPoolDescriptionByIDParams struct {

	/*PoolID*/
	PoolID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the get pool description by ID params
func (o *GetPoolDescriptionByIDParams) WithTimeout(timeout time.Duration) *GetPoolDescriptionByIDParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get pool description by ID params
func (o *GetPoolDescriptionByIDParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get pool description by ID params
func (o *GetPoolDescriptionByIDParams) WithContext(ctx context.Context) *GetPoolDescriptionByIDParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get pool description by ID params
func (o *GetPoolDescriptionByIDParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get pool description by ID params
func (o *GetPoolDescriptionByIDParams) WithHTTPClient(client *http.Client) *GetPoolDescriptionByIDParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get pool description by ID params
func (o *GetPoolDescriptionByIDParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithPoolID adds the poolID to the get pool description by ID params
func (o *GetPoolDescriptionByIDParams) WithPoolID(poolID string) *GetPoolDescriptionByIDParams {
	o.SetPoolID(poolID)
	return o
}

// SetPoolID adds the poolId to the get pool description by ID params
func (o *GetPoolDescriptionByIDParams) SetPoolID(poolID string) {
	o.PoolID = poolID
}

// WriteToRequest writes these params to a swagger request
func (o *GetPoolDescriptionByIDParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param pool_id
	if err := r.SetPathParam("pool_id", o.PoolID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
