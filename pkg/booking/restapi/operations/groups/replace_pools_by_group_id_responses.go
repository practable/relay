// Code generated by go-swagger; DO NOT EDIT.

package groups

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/timdrysdale/relay/pkg/booking/models"
)

// ReplacePoolsByGroupIDOKCode is the HTTP code returned for type ReplacePoolsByGroupIDOK
const ReplacePoolsByGroupIDOKCode int = 200

/*ReplacePoolsByGroupIDOK replace pools by group Id o k

swagger:response replacePoolsByGroupIdOK
*/
type ReplacePoolsByGroupIDOK struct {

	/*
	  In: Body
	*/
	Payload models.IDList `json:"body,omitempty"`
}

// NewReplacePoolsByGroupIDOK creates ReplacePoolsByGroupIDOK with default headers values
func NewReplacePoolsByGroupIDOK() *ReplacePoolsByGroupIDOK {

	return &ReplacePoolsByGroupIDOK{}
}

// WithPayload adds the payload to the replace pools by group Id o k response
func (o *ReplacePoolsByGroupIDOK) WithPayload(payload models.IDList) *ReplacePoolsByGroupIDOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the replace pools by group Id o k response
func (o *ReplacePoolsByGroupIDOK) SetPayload(payload models.IDList) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *ReplacePoolsByGroupIDOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	payload := o.Payload
	if payload == nil {
		// return empty array
		payload = models.IDList{}
	}

	if err := producer.Produce(rw, payload); err != nil {
		panic(err) // let the recovery middleware deal with this
	}
}

// ReplacePoolsByGroupIDUnauthorizedCode is the HTTP code returned for type ReplacePoolsByGroupIDUnauthorized
const ReplacePoolsByGroupIDUnauthorizedCode int = 401

/*ReplacePoolsByGroupIDUnauthorized Unauthorized

swagger:response replacePoolsByGroupIdUnauthorized
*/
type ReplacePoolsByGroupIDUnauthorized struct {

	/*
	  In: Body
	*/
	Payload interface{} `json:"body,omitempty"`
}

// NewReplacePoolsByGroupIDUnauthorized creates ReplacePoolsByGroupIDUnauthorized with default headers values
func NewReplacePoolsByGroupIDUnauthorized() *ReplacePoolsByGroupIDUnauthorized {

	return &ReplacePoolsByGroupIDUnauthorized{}
}

// WithPayload adds the payload to the replace pools by group Id unauthorized response
func (o *ReplacePoolsByGroupIDUnauthorized) WithPayload(payload interface{}) *ReplacePoolsByGroupIDUnauthorized {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the replace pools by group Id unauthorized response
func (o *ReplacePoolsByGroupIDUnauthorized) SetPayload(payload interface{}) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *ReplacePoolsByGroupIDUnauthorized) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(401)
	payload := o.Payload
	if err := producer.Produce(rw, payload); err != nil {
		panic(err) // let the recovery middleware deal with this
	}
}

// ReplacePoolsByGroupIDNotFoundCode is the HTTP code returned for type ReplacePoolsByGroupIDNotFound
const ReplacePoolsByGroupIDNotFoundCode int = 404

/*ReplacePoolsByGroupIDNotFound Not Found

swagger:response replacePoolsByGroupIdNotFound
*/
type ReplacePoolsByGroupIDNotFound struct {
}

// NewReplacePoolsByGroupIDNotFound creates ReplacePoolsByGroupIDNotFound with default headers values
func NewReplacePoolsByGroupIDNotFound() *ReplacePoolsByGroupIDNotFound {

	return &ReplacePoolsByGroupIDNotFound{}
}

// WriteResponse to the client
func (o *ReplacePoolsByGroupIDNotFound) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.Header().Del(runtime.HeaderContentType) //Remove Content-Type on empty responses

	rw.WriteHeader(404)
}

// ReplacePoolsByGroupIDInternalServerErrorCode is the HTTP code returned for type ReplacePoolsByGroupIDInternalServerError
const ReplacePoolsByGroupIDInternalServerErrorCode int = 500

/*ReplacePoolsByGroupIDInternalServerError replace pools by group Id internal server error

swagger:response replacePoolsByGroupIdInternalServerError
*/
type ReplacePoolsByGroupIDInternalServerError struct {

	/*
	  In: Body
	*/
	Payload interface{} `json:"body,omitempty"`
}

// NewReplacePoolsByGroupIDInternalServerError creates ReplacePoolsByGroupIDInternalServerError with default headers values
func NewReplacePoolsByGroupIDInternalServerError() *ReplacePoolsByGroupIDInternalServerError {

	return &ReplacePoolsByGroupIDInternalServerError{}
}

// WithPayload adds the payload to the replace pools by group Id internal server error response
func (o *ReplacePoolsByGroupIDInternalServerError) WithPayload(payload interface{}) *ReplacePoolsByGroupIDInternalServerError {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the replace pools by group Id internal server error response
func (o *ReplacePoolsByGroupIDInternalServerError) SetPayload(payload interface{}) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *ReplacePoolsByGroupIDInternalServerError) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(500)
	payload := o.Payload
	if err := producer.Produce(rw, payload); err != nil {
		panic(err) // let the recovery middleware deal with this
	}
}
