// Code generated by go-swagger; DO NOT EDIT.

package groups

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"
)

// GetGroupIDByNameOKCode is the HTTP code returned for type GetGroupIDByNameOK
const GetGroupIDByNameOKCode int = 200

/*GetGroupIDByNameOK get group Id by name o k

swagger:response getGroupIdByNameOK
*/
type GetGroupIDByNameOK struct {

	/*
	  In: Body
	*/
	Payload []string `json:"body,omitempty"`
}

// NewGetGroupIDByNameOK creates GetGroupIDByNameOK with default headers values
func NewGetGroupIDByNameOK() *GetGroupIDByNameOK {

	return &GetGroupIDByNameOK{}
}

// WithPayload adds the payload to the get group Id by name o k response
func (o *GetGroupIDByNameOK) WithPayload(payload []string) *GetGroupIDByNameOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get group Id by name o k response
func (o *GetGroupIDByNameOK) SetPayload(payload []string) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetGroupIDByNameOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	payload := o.Payload
	if payload == nil {
		// return empty array
		payload = make([]string, 0, 50)
	}

	if err := producer.Produce(rw, payload); err != nil {
		panic(err) // let the recovery middleware deal with this
	}
}

// GetGroupIDByNameUnauthorizedCode is the HTTP code returned for type GetGroupIDByNameUnauthorized
const GetGroupIDByNameUnauthorizedCode int = 401

/*GetGroupIDByNameUnauthorized Unauthorized

swagger:response getGroupIdByNameUnauthorized
*/
type GetGroupIDByNameUnauthorized struct {

	/*
	  In: Body
	*/
	Payload interface{} `json:"body,omitempty"`
}

// NewGetGroupIDByNameUnauthorized creates GetGroupIDByNameUnauthorized with default headers values
func NewGetGroupIDByNameUnauthorized() *GetGroupIDByNameUnauthorized {

	return &GetGroupIDByNameUnauthorized{}
}

// WithPayload adds the payload to the get group Id by name unauthorized response
func (o *GetGroupIDByNameUnauthorized) WithPayload(payload interface{}) *GetGroupIDByNameUnauthorized {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get group Id by name unauthorized response
func (o *GetGroupIDByNameUnauthorized) SetPayload(payload interface{}) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetGroupIDByNameUnauthorized) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(401)
	payload := o.Payload
	if err := producer.Produce(rw, payload); err != nil {
		panic(err) // let the recovery middleware deal with this
	}
}

// GetGroupIDByNameInternalServerErrorCode is the HTTP code returned for type GetGroupIDByNameInternalServerError
const GetGroupIDByNameInternalServerErrorCode int = 500

/*GetGroupIDByNameInternalServerError get group Id by name internal server error

swagger:response getGroupIdByNameInternalServerError
*/
type GetGroupIDByNameInternalServerError struct {

	/*
	  In: Body
	*/
	Payload interface{} `json:"body,omitempty"`
}

// NewGetGroupIDByNameInternalServerError creates GetGroupIDByNameInternalServerError with default headers values
func NewGetGroupIDByNameInternalServerError() *GetGroupIDByNameInternalServerError {

	return &GetGroupIDByNameInternalServerError{}
}

// WithPayload adds the payload to the get group Id by name internal server error response
func (o *GetGroupIDByNameInternalServerError) WithPayload(payload interface{}) *GetGroupIDByNameInternalServerError {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get group Id by name internal server error response
func (o *GetGroupIDByNameInternalServerError) SetPayload(payload interface{}) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetGroupIDByNameInternalServerError) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(500)
	payload := o.Payload
	if err := producer.Produce(rw, payload); err != nil {
		panic(err) // let the recovery middleware deal with this
	}
}