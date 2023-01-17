// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/practable/relay/internal/access/models"
)

// DenyNoContentCode is the HTTP code returned for type DenyNoContent
const DenyNoContentCode int = 204

/*DenyNoContent The bid was denied successfully.

swagger:response denyNoContent
*/
type DenyNoContent struct {

	/*
	  In: Body
	*/
	Payload interface{} `json:"body,omitempty"`
}

// NewDenyNoContent creates DenyNoContent with default headers values
func NewDenyNoContent() *DenyNoContent {

	return &DenyNoContent{}
}

// WithPayload adds the payload to the deny no content response
func (o *DenyNoContent) WithPayload(payload interface{}) *DenyNoContent {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the deny no content response
func (o *DenyNoContent) SetPayload(payload interface{}) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *DenyNoContent) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(204)
	payload := o.Payload
	if err := producer.Produce(rw, payload); err != nil {
		panic(err) // let the recovery middleware deal with this
	}
}

// DenyBadRequestCode is the HTTP code returned for type DenyBadRequest
const DenyBadRequestCode int = 400

/*DenyBadRequest BadRequest

swagger:response denyBadRequest
*/
type DenyBadRequest struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewDenyBadRequest creates DenyBadRequest with default headers values
func NewDenyBadRequest() *DenyBadRequest {

	return &DenyBadRequest{}
}

// WithPayload adds the payload to the deny bad request response
func (o *DenyBadRequest) WithPayload(payload *models.Error) *DenyBadRequest {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the deny bad request response
func (o *DenyBadRequest) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *DenyBadRequest) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(400)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// DenyUnauthorizedCode is the HTTP code returned for type DenyUnauthorized
const DenyUnauthorizedCode int = 401

/*DenyUnauthorized Unauthorized

swagger:response denyUnauthorized
*/
type DenyUnauthorized struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewDenyUnauthorized creates DenyUnauthorized with default headers values
func NewDenyUnauthorized() *DenyUnauthorized {

	return &DenyUnauthorized{}
}

// WithPayload adds the payload to the deny unauthorized response
func (o *DenyUnauthorized) WithPayload(payload *models.Error) *DenyUnauthorized {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the deny unauthorized response
func (o *DenyUnauthorized) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *DenyUnauthorized) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(401)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}
