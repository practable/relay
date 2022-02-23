// Code generated by go-swagger; DO NOT EDIT.

package pools

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"
)

// DeleteActivityByIDUnauthorizedCode is the HTTP code returned for type DeleteActivityByIDUnauthorized
const DeleteActivityByIDUnauthorizedCode int = 401

/*DeleteActivityByIDUnauthorized Unauthorized

swagger:response deleteActivityByIdUnauthorized
*/
type DeleteActivityByIDUnauthorized struct {

	/*
	  In: Body
	*/
	Payload interface{} `json:"body,omitempty"`
}

// NewDeleteActivityByIDUnauthorized creates DeleteActivityByIDUnauthorized with default headers values
func NewDeleteActivityByIDUnauthorized() *DeleteActivityByIDUnauthorized {

	return &DeleteActivityByIDUnauthorized{}
}

// WithPayload adds the payload to the delete activity by Id unauthorized response
func (o *DeleteActivityByIDUnauthorized) WithPayload(payload interface{}) *DeleteActivityByIDUnauthorized {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the delete activity by Id unauthorized response
func (o *DeleteActivityByIDUnauthorized) SetPayload(payload interface{}) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *DeleteActivityByIDUnauthorized) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(401)
	payload := o.Payload
	if err := producer.Produce(rw, payload); err != nil {
		panic(err) // let the recovery middleware deal with this
	}
}

// DeleteActivityByIDNotFoundCode is the HTTP code returned for type DeleteActivityByIDNotFound
const DeleteActivityByIDNotFoundCode int = 404

/*DeleteActivityByIDNotFound Not Found

swagger:response deleteActivityByIdNotFound
*/
type DeleteActivityByIDNotFound struct {

	/*
	  In: Body
	*/
	Payload interface{} `json:"body,omitempty"`
}

// NewDeleteActivityByIDNotFound creates DeleteActivityByIDNotFound with default headers values
func NewDeleteActivityByIDNotFound() *DeleteActivityByIDNotFound {

	return &DeleteActivityByIDNotFound{}
}

// WithPayload adds the payload to the delete activity by Id not found response
func (o *DeleteActivityByIDNotFound) WithPayload(payload interface{}) *DeleteActivityByIDNotFound {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the delete activity by Id not found response
func (o *DeleteActivityByIDNotFound) SetPayload(payload interface{}) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *DeleteActivityByIDNotFound) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(404)
	payload := o.Payload
	if err := producer.Produce(rw, payload); err != nil {
		panic(err) // let the recovery middleware deal with this
	}
}

// DeleteActivityByIDInternalServerErrorCode is the HTTP code returned for type DeleteActivityByIDInternalServerError
const DeleteActivityByIDInternalServerErrorCode int = 500

/*DeleteActivityByIDInternalServerError Internal Error

swagger:response deleteActivityByIdInternalServerError
*/
type DeleteActivityByIDInternalServerError struct {

	/*
	  In: Body
	*/
	Payload interface{} `json:"body,omitempty"`
}

// NewDeleteActivityByIDInternalServerError creates DeleteActivityByIDInternalServerError with default headers values
func NewDeleteActivityByIDInternalServerError() *DeleteActivityByIDInternalServerError {

	return &DeleteActivityByIDInternalServerError{}
}

// WithPayload adds the payload to the delete activity by Id internal server error response
func (o *DeleteActivityByIDInternalServerError) WithPayload(payload interface{}) *DeleteActivityByIDInternalServerError {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the delete activity by Id internal server error response
func (o *DeleteActivityByIDInternalServerError) SetPayload(payload interface{}) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *DeleteActivityByIDInternalServerError) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(500)
	payload := o.Payload
	if err := producer.Produce(rw, payload); err != nil {
		panic(err) // let the recovery middleware deal with this
	}
}