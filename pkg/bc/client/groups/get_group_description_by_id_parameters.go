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
)

// NewGetGroupDescriptionByIDParams creates a new GetGroupDescriptionByIDParams object
// with the default values initialized.
func NewGetGroupDescriptionByIDParams() *GetGroupDescriptionByIDParams {
	var ()
	return &GetGroupDescriptionByIDParams{

		timeout: cr.DefaultTimeout,
	}
}

// NewGetGroupDescriptionByIDParamsWithTimeout creates a new GetGroupDescriptionByIDParams object
// with the default values initialized, and the ability to set a timeout on a request
func NewGetGroupDescriptionByIDParamsWithTimeout(timeout time.Duration) *GetGroupDescriptionByIDParams {
	var ()
	return &GetGroupDescriptionByIDParams{

		timeout: timeout,
	}
}

// NewGetGroupDescriptionByIDParamsWithContext creates a new GetGroupDescriptionByIDParams object
// with the default values initialized, and the ability to set a context for a request
func NewGetGroupDescriptionByIDParamsWithContext(ctx context.Context) *GetGroupDescriptionByIDParams {
	var ()
	return &GetGroupDescriptionByIDParams{

		Context: ctx,
	}
}

// NewGetGroupDescriptionByIDParamsWithHTTPClient creates a new GetGroupDescriptionByIDParams object
// with the default values initialized, and the ability to set a custom HTTPClient for a request
func NewGetGroupDescriptionByIDParamsWithHTTPClient(client *http.Client) *GetGroupDescriptionByIDParams {
	var ()
	return &GetGroupDescriptionByIDParams{
		HTTPClient: client,
	}
}

/*GetGroupDescriptionByIDParams contains all the parameters to send to the API endpoint
for the get group description by ID operation typically these are written to a http.Request
*/
type GetGroupDescriptionByIDParams struct {

	/*GroupID*/
	GroupID string

	timeout    time.Duration
	Context    context.Context
	HTTPClient *http.Client
}

// WithTimeout adds the timeout to the get group description by ID params
func (o *GetGroupDescriptionByIDParams) WithTimeout(timeout time.Duration) *GetGroupDescriptionByIDParams {
	o.SetTimeout(timeout)
	return o
}

// SetTimeout adds the timeout to the get group description by ID params
func (o *GetGroupDescriptionByIDParams) SetTimeout(timeout time.Duration) {
	o.timeout = timeout
}

// WithContext adds the context to the get group description by ID params
func (o *GetGroupDescriptionByIDParams) WithContext(ctx context.Context) *GetGroupDescriptionByIDParams {
	o.SetContext(ctx)
	return o
}

// SetContext adds the context to the get group description by ID params
func (o *GetGroupDescriptionByIDParams) SetContext(ctx context.Context) {
	o.Context = ctx
}

// WithHTTPClient adds the HTTPClient to the get group description by ID params
func (o *GetGroupDescriptionByIDParams) WithHTTPClient(client *http.Client) *GetGroupDescriptionByIDParams {
	o.SetHTTPClient(client)
	return o
}

// SetHTTPClient adds the HTTPClient to the get group description by ID params
func (o *GetGroupDescriptionByIDParams) SetHTTPClient(client *http.Client) {
	o.HTTPClient = client
}

// WithGroupID adds the groupID to the get group description by ID params
func (o *GetGroupDescriptionByIDParams) WithGroupID(groupID string) *GetGroupDescriptionByIDParams {
	o.SetGroupID(groupID)
	return o
}

// SetGroupID adds the groupId to the get group description by ID params
func (o *GetGroupDescriptionByIDParams) SetGroupID(groupID string) {
	o.GroupID = groupID
}

// WriteToRequest writes these params to a swagger request
func (o *GetGroupDescriptionByIDParams) WriteToRequest(r runtime.ClientRequest, reg strfmt.Registry) error {

	if err := r.SetTimeout(o.timeout); err != nil {
		return err
	}
	var res []error

	// path param group_id
	if err := r.SetPathParam("group_id", o.GroupID); err != nil {
		return err
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
