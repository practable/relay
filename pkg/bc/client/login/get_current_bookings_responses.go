// Code generated by go-swagger; DO NOT EDIT.

package login

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/timdrysdale/relay/pkg/bc/models"
)

// GetCurrentBookingsReader is a Reader for the GetCurrentBookings structure.
type GetCurrentBookingsReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetCurrentBookingsReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetCurrentBookingsOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewGetCurrentBookingsUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 404:
		result := NewGetCurrentBookingsNotFound()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 500:
		result := NewGetCurrentBookingsInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewGetCurrentBookingsOK creates a GetCurrentBookingsOK with default headers values
func NewGetCurrentBookingsOK() *GetCurrentBookingsOK {
	return &GetCurrentBookingsOK{}
}

/* GetCurrentBookingsOK describes a response with status code 200, with default header values.

GetCurrentBookingsOK get current bookings o k
*/
type GetCurrentBookingsOK struct {
	Payload *models.Bookings
}

func (o *GetCurrentBookingsOK) Error() string {
	return fmt.Sprintf("[GET /login][%d] getCurrentBookingsOK  %+v", 200, o.Payload)
}
func (o *GetCurrentBookingsOK) GetPayload() *models.Bookings {
	return o.Payload
}

func (o *GetCurrentBookingsOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Bookings)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetCurrentBookingsUnauthorized creates a GetCurrentBookingsUnauthorized with default headers values
func NewGetCurrentBookingsUnauthorized() *GetCurrentBookingsUnauthorized {
	return &GetCurrentBookingsUnauthorized{}
}

/* GetCurrentBookingsUnauthorized describes a response with status code 401, with default header values.

Unauthorized
*/
type GetCurrentBookingsUnauthorized struct {
	Payload interface{}
}

func (o *GetCurrentBookingsUnauthorized) Error() string {
	return fmt.Sprintf("[GET /login][%d] getCurrentBookingsUnauthorized  %+v", 401, o.Payload)
}
func (o *GetCurrentBookingsUnauthorized) GetPayload() interface{} {
	return o.Payload
}

func (o *GetCurrentBookingsUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetCurrentBookingsNotFound creates a GetCurrentBookingsNotFound with default headers values
func NewGetCurrentBookingsNotFound() *GetCurrentBookingsNotFound {
	return &GetCurrentBookingsNotFound{}
}

/* GetCurrentBookingsNotFound describes a response with status code 404, with default header values.

Not Found
*/
type GetCurrentBookingsNotFound struct {
	Payload interface{}
}

func (o *GetCurrentBookingsNotFound) Error() string {
	return fmt.Sprintf("[GET /login][%d] getCurrentBookingsNotFound  %+v", 404, o.Payload)
}
func (o *GetCurrentBookingsNotFound) GetPayload() interface{} {
	return o.Payload
}

func (o *GetCurrentBookingsNotFound) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetCurrentBookingsInternalServerError creates a GetCurrentBookingsInternalServerError with default headers values
func NewGetCurrentBookingsInternalServerError() *GetCurrentBookingsInternalServerError {
	return &GetCurrentBookingsInternalServerError{}
}

/* GetCurrentBookingsInternalServerError describes a response with status code 500, with default header values.

Internal Error
*/
type GetCurrentBookingsInternalServerError struct {
	Payload interface{}
}

func (o *GetCurrentBookingsInternalServerError) Error() string {
	return fmt.Sprintf("[GET /login][%d] getCurrentBookingsInternalServerError  %+v", 500, o.Payload)
}
func (o *GetCurrentBookingsInternalServerError) GetPayload() interface{} {
	return o.Payload
}

func (o *GetCurrentBookingsInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
