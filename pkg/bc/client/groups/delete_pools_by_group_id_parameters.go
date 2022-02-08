// Code generated by go-swagger; DO NOT EDIT.

package groups

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

	"github.com/timdrysdale/relay/pkg/bc/models"
)

// NewDeletePoolsByGroupIDParams creates a new DeletePoolsByGroupIDParams object,
// with the default timeout for this client.
//
// Default values are not hydrated, since defaults are normally applied by the API server side.
//
// To enforce default values in parameter, use SetDefaults or WithDefaults.
func NewDeletePoolsByGroupIDParams() *DeletePoolsByGroupIDParams {
	return &DeletePoolsByGroupIDParams{
		timeout: cr.DefaultTimeout,
	}
}

// NewDeletePoolsByGroupIDParamsWithTimeout creates a new DeletePoolsByGroupIDParams object
// with the ability to set a timeout on a request.
func NewDeletePoolsByGroupIDParamsWithTimeout(timeout time.Duration) *DeletePoolsByGroupIDParams {
	return &DeletePoolsByGroupIDParams{
		timeout: timeout,
	}
}

// NewDeletePoolsByGroupIDParamsWithContext creates a new DeletePoolsByGroupIDParams object
// with the ability to set a context for a request.
func NewDeletePoolsByGroupIDParamsWithContext(ctx context.Context) *DeletePoolsByGroupIDParams {
	return &DeletePoolsByGroupIDParams{
		Context: ctx,
	}
}

// NewDeletePoolsByGroupIDParamsWithHTTPClient creates a new DeletePoolsByGroupIDParams object
// with the ability to set a custom HTTPClient for a request.
func NewDeletePoolsByGroupIDParamsWithHTTPClient(client *http.Client) *DeletePoolsByGroupIDParams {
	return &DeletePoolsByGroupIDParams{
		HTTPClient: client,
	}
}

/* DeletePoolsByGroupIDParams contains all the parameters to send to the API endpoint
   for the delete pools by group ID operation.

   Typically these are written to a http.Request.
*/
type DeletePoolsByGroupIDParams struct {

	// GroupID.
	GroupID string

	// Pools.
	Pools models.IDList

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithDefaults hydrates default values in the delete pools by group ID params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeletePoolsByGroupIDParams) WithDefaults() *DeletePoolsByGroupIDParams {
	o.SetDefaults()
	return o
}

// SetDefaults hydrates default values in the delete pools by group ID params (not the query body).
//
// All values with no default are reset to their zero value.
func (o *DeletePoolsByGroupIDParams) SetDefaults() {
	// no default values defined for this parameter
}

// WithTimeout adds the timeout to the delete pools by group ID params
func (o *DeletePoolsByGroupIDParams) WithTimeout(timeout time.Duration) *DeletePoolsByGroupIDParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the delete pools by group ID params
func (o *DeletePoolsByGroupIDParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the delete pools by group ID params
func (o *DeletePoolsByGroupIDParams) WithContext(ctx context.Context) *DeletePoolsByGroupIDParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the delete pools by group ID params
func (o *DeletePoolsByGroupIDParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the delete pools by group ID params
func (o *DeletePoolsByGroupIDParams) WithHTTPClient(client *http.Client) *DeletePoolsByGroupIDParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the delete pools by group ID params
func (o *DeletePoolsByGroupIDParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithGroupID adds the groupID to the delete pools by group ID params
func (o *DeletePoolsByGroupIDParams) WithGroupID(groupID string) *DeletePoolsByGroupIDParams {
	o.SetGroupID(groupID)
	return o
}

// SetGroupID adds the groupId to the delete pools by group ID params
func (o *DeletePoolsByGroupIDParams) SetGroupID(groupID string) {
	o.GroupID = groupID
}

// WithPools adds the pools to the delete pools by group ID params
func (o *DeletePoolsByGroupIDParams) WithPools(pools models.IDList) *DeletePoolsByGroupIDParams {
	o.SetPools(pools)
	return o
}

// SetPools adds the pools to the delete pools by group ID params
func (o *DeletePoolsByGroupIDParams) SetPools(pools models.IDList) {
	o.Pools = pools
}

// WriteToRequest writes these params to a swagger request
func (o *DeletePoolsByGroupIDParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param group_id
	if err := r.SetPathParam("group_id", o.GroupID); err != nil {
		return err
	}
	if o.Pools != nil {
		if err := r.SetBodyParam(o.Pools); err != nil {
			return err
		}
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
