// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/practable/relay/internal/access/models"
)

// ListAllowedOKCode is the HTTP code returned for type ListAllowedOK
const ListAllowedOKCode int = 200

/*ListAllowedOK Current or recently in-use allowed bids

swagger:response listAllowedOK
*/
type ListAllowedOK struct {

	/*
	  In: Body
	*/
	Payload *models.BookingIDs `json:"body,omitempty"`
}

// NewListAllowedOK creates ListAllowedOK with default headers values
func NewListAllowedOK() *ListAllowedOK {

	return &ListAllowedOK{}
}

// WithPayload adds the payload to the list allowed o k response
func (o *ListAllowedOK) WithPayload(payload *models.BookingIDs) *ListAllowedOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the list allowed o k response
func (o *ListAllowedOK) SetPayload(payload *models.BookingIDs) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *ListAllowedOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// ListAllowedUnauthorizedCode is the HTTP code returned for type ListAllowedUnauthorized
const ListAllowedUnauthorizedCode int = 401

/*ListAllowedUnauthorized Unauthorized

swagger:response listAllowedUnauthorized
*/
type ListAllowedUnauthorized struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewListAllowedUnauthorized creates ListAllowedUnauthorized with default headers values
func NewListAllowedUnauthorized() *ListAllowedUnauthorized {

	return &ListAllowedUnauthorized{}
}

// WithPayload adds the payload to the list allowed unauthorized response
func (o *ListAllowedUnauthorized) WithPayload(payload *models.Error) *ListAllowedUnauthorized {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the list allowed unauthorized response
func (o *ListAllowedUnauthorized) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *ListAllowedUnauthorized) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(401)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}
