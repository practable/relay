// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/practable/relay/internal/access/models"
)

// AllowNoContentCode is the HTTP code returned for type AllowNoContent
const AllowNoContentCode int = 204

/*AllowNoContent The bid was allowed successfully.

swagger:response allowNoContent
*/
type AllowNoContent struct {
}

// NewAllowNoContent creates AllowNoContent with default headers values
func NewAllowNoContent() *AllowNoContent {

	return &AllowNoContent{}
}

// WriteResponse to the client
func (o *AllowNoContent) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.Header().Del(runtime.HeaderContentType) //Remove Content-Type on empty responses

	rw.WriteHeader(204)
}

// AllowBadRequestCode is the HTTP code returned for type AllowBadRequest
const AllowBadRequestCode int = 400

/*AllowBadRequest BadRequest

swagger:response allowBadRequest
*/
type AllowBadRequest struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewAllowBadRequest creates AllowBadRequest with default headers values
func NewAllowBadRequest() *AllowBadRequest {

	return &AllowBadRequest{}
}

// WithPayload adds the payload to the allow bad request response
func (o *AllowBadRequest) WithPayload(payload *models.Error) *AllowBadRequest {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the allow bad request response
func (o *AllowBadRequest) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *AllowBadRequest) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(400)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// AllowUnauthorizedCode is the HTTP code returned for type AllowUnauthorized
const AllowUnauthorizedCode int = 401

/*AllowUnauthorized Unauthorized

swagger:response allowUnauthorized
*/
type AllowUnauthorized struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewAllowUnauthorized creates AllowUnauthorized with default headers values
func NewAllowUnauthorized() *AllowUnauthorized {

	return &AllowUnauthorized{}
}

// WithPayload adds the payload to the allow unauthorized response
func (o *AllowUnauthorized) WithPayload(payload *models.Error) *AllowUnauthorized {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the allow unauthorized response
func (o *AllowUnauthorized) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *AllowUnauthorized) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(401)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}
