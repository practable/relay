// Code generated by go-swagger; DO NOT EDIT.

package admin

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/timdrysdale/relay/pkg/bc/models"
)

// GetStoreStatusReader is a Reader for the GetStoreStatus structure.
type GetStoreStatusReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *GetStoreStatusReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewGetStoreStatusOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewGetStoreStatusUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 500:
		result := NewGetStoreStatusInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewGetStoreStatusOK creates a GetStoreStatusOK with default headers values
func NewGetStoreStatusOK() *GetStoreStatusOK {
	return &GetStoreStatusOK{}
}

/* GetStoreStatusOK describes a response with status code 200, with default header values.

GetStoreStatusOK get store status o k
*/
type GetStoreStatusOK struct {
	Payload *models.StoreStatus
}

func (o *GetStoreStatusOK) Error() string {
	return fmt.Sprintf("[GET /admin/status][%d] getStoreStatusOK  %+v", 200, o.Payload)
}
func (o *GetStoreStatusOK) GetPayload() *models.StoreStatus {
	return o.Payload
}

func (o *GetStoreStatusOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.StoreStatus)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetStoreStatusUnauthorized creates a GetStoreStatusUnauthorized with default headers values
func NewGetStoreStatusUnauthorized() *GetStoreStatusUnauthorized {
	return &GetStoreStatusUnauthorized{}
}

/* GetStoreStatusUnauthorized describes a response with status code 401, with default header values.

Unauthorized
*/
type GetStoreStatusUnauthorized struct {
	Payload interface{}
}

func (o *GetStoreStatusUnauthorized) Error() string {
	return fmt.Sprintf("[GET /admin/status][%d] getStoreStatusUnauthorized  %+v", 401, o.Payload)
}
func (o *GetStoreStatusUnauthorized) GetPayload() interface{} {
	return o.Payload
}

func (o *GetStoreStatusUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewGetStoreStatusInternalServerError creates a GetStoreStatusInternalServerError with default headers values
func NewGetStoreStatusInternalServerError() *GetStoreStatusInternalServerError {
	return &GetStoreStatusInternalServerError{}
}

/* GetStoreStatusInternalServerError describes a response with status code 500, with default header values.

GetStoreStatusInternalServerError get store status internal server error
*/
type GetStoreStatusInternalServerError struct {
	Payload interface{}
}

func (o *GetStoreStatusInternalServerError) Error() string {
	return fmt.Sprintf("[GET /admin/status][%d] getStoreStatusInternalServerError  %+v", 500, o.Payload)
}
func (o *GetStoreStatusInternalServerError) GetPayload() interface{} {
	return o.Payload
}

func (o *GetStoreStatusInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
