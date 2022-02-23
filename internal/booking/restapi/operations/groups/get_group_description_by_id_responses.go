// Code generated by go-swagger; DO NOT EDIT.

package groups

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/practable/relay/internal/booking/models"
)

// GetGroupDescriptionByIDOKCode is the HTTP code returned for type GetGroupDescriptionByIDOK
const GetGroupDescriptionByIDOKCode int = 200

/*GetGroupDescriptionByIDOK get group description by Id o k

swagger:response getGroupDescriptionByIdOK
*/
type GetGroupDescriptionByIDOK struct {

	/*
	  In: Body
	*/
	Payload *models.Description `json:"body,omitempty"`
}

// NewGetGroupDescriptionByIDOK creates GetGroupDescriptionByIDOK with default headers values
func NewGetGroupDescriptionByIDOK() *GetGroupDescriptionByIDOK {

	return &GetGroupDescriptionByIDOK{}
}

// WithPayload adds the payload to the get group description by Id o k response
func (o *GetGroupDescriptionByIDOK) WithPayload(payload *models.Description) *GetGroupDescriptionByIDOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get group description by Id o k response
func (o *GetGroupDescriptionByIDOK) SetPayload(payload *models.Description) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetGroupDescriptionByIDOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// GetGroupDescriptionByIDUnauthorizedCode is the HTTP code returned for type GetGroupDescriptionByIDUnauthorized
const GetGroupDescriptionByIDUnauthorizedCode int = 401

/*GetGroupDescriptionByIDUnauthorized Unauthorized

swagger:response getGroupDescriptionByIdUnauthorized
*/
type GetGroupDescriptionByIDUnauthorized struct {

	/*
	  In: Body
	*/
	Payload interface{} `json:"body,omitempty"`
}

// NewGetGroupDescriptionByIDUnauthorized creates GetGroupDescriptionByIDUnauthorized with default headers values
func NewGetGroupDescriptionByIDUnauthorized() *GetGroupDescriptionByIDUnauthorized {

	return &GetGroupDescriptionByIDUnauthorized{}
}

// WithPayload adds the payload to the get group description by Id unauthorized response
func (o *GetGroupDescriptionByIDUnauthorized) WithPayload(payload interface{}) *GetGroupDescriptionByIDUnauthorized {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get group description by Id unauthorized response
func (o *GetGroupDescriptionByIDUnauthorized) SetPayload(payload interface{}) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetGroupDescriptionByIDUnauthorized) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(401)
	payload := o.Payload
	if err := producer.Produce(rw, payload); err != nil {
		panic(err) // let the recovery middleware deal with this
	}
}

// GetGroupDescriptionByIDNotFoundCode is the HTTP code returned for type GetGroupDescriptionByIDNotFound
const GetGroupDescriptionByIDNotFoundCode int = 404

/*GetGroupDescriptionByIDNotFound Not Found

swagger:response getGroupDescriptionByIdNotFound
*/
type GetGroupDescriptionByIDNotFound struct {

	/*
	  In: Body
	*/
	Payload interface{} `json:"body,omitempty"`
}

// NewGetGroupDescriptionByIDNotFound creates GetGroupDescriptionByIDNotFound with default headers values
func NewGetGroupDescriptionByIDNotFound() *GetGroupDescriptionByIDNotFound {

	return &GetGroupDescriptionByIDNotFound{}
}

// WithPayload adds the payload to the get group description by Id not found response
func (o *GetGroupDescriptionByIDNotFound) WithPayload(payload interface{}) *GetGroupDescriptionByIDNotFound {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get group description by Id not found response
func (o *GetGroupDescriptionByIDNotFound) SetPayload(payload interface{}) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetGroupDescriptionByIDNotFound) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(404)
	payload := o.Payload
	if err := producer.Produce(rw, payload); err != nil {
		panic(err) // let the recovery middleware deal with this
	}
}

// GetGroupDescriptionByIDInternalServerErrorCode is the HTTP code returned for type GetGroupDescriptionByIDInternalServerError
const GetGroupDescriptionByIDInternalServerErrorCode int = 500

/*GetGroupDescriptionByIDInternalServerError get group description by Id internal server error

swagger:response getGroupDescriptionByIdInternalServerError
*/
type GetGroupDescriptionByIDInternalServerError struct {

	/*
	  In: Body
	*/
	Payload interface{} `json:"body,omitempty"`
}

// NewGetGroupDescriptionByIDInternalServerError creates GetGroupDescriptionByIDInternalServerError with default headers values
func NewGetGroupDescriptionByIDInternalServerError() *GetGroupDescriptionByIDInternalServerError {

	return &GetGroupDescriptionByIDInternalServerError{}
}

// WithPayload adds the payload to the get group description by Id internal server error response
func (o *GetGroupDescriptionByIDInternalServerError) WithPayload(payload interface{}) *GetGroupDescriptionByIDInternalServerError {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get group description by Id internal server error response
func (o *GetGroupDescriptionByIDInternalServerError) SetPayload(payload interface{}) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetGroupDescriptionByIDInternalServerError) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(500)
	payload := o.Payload
	if err := producer.Produce(rw, payload); err != nil {
		panic(err) // let the recovery middleware deal with this
	}
}