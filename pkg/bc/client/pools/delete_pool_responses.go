// Code generated by go-swagger; DO NOT EDIT.

package pools

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"
)

// DeletePoolReader is a Reader for the DeletePool structure.
type DeletePoolReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeletePoolReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 401:
		result := NewDeletePoolUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 404:
		result := NewDeletePoolNotFound()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 500:
		result := NewDeletePoolInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewDeletePoolUnauthorized creates a DeletePoolUnauthorized with default headers values
func NewDeletePoolUnauthorized() *DeletePoolUnauthorized {
	return &DeletePoolUnauthorized{}
}

/* DeletePoolUnauthorized describes a response with status code 401, with default header values.

Unauthorized
*/
type DeletePoolUnauthorized struct {
	Payload interface{}
}

func (o *DeletePoolUnauthorized) Error() string {
	return fmt.Sprintf("[DELETE /pools/{pool_id}][%d] deletePoolUnauthorized  %+v", 401, o.Payload)
}
func (o *DeletePoolUnauthorized) GetPayload() interface{} {
	return o.Payload
}

func (o *DeletePoolUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewDeletePoolNotFound creates a DeletePoolNotFound with default headers values
func NewDeletePoolNotFound() *DeletePoolNotFound {
	return &DeletePoolNotFound{}
}

/* DeletePoolNotFound describes a response with status code 404, with default header values.

Not Found
*/
type DeletePoolNotFound struct {
	Payload interface{}
}

func (o *DeletePoolNotFound) Error() string {
	return fmt.Sprintf("[DELETE /pools/{pool_id}][%d] deletePoolNotFound  %+v", 404, o.Payload)
}
func (o *DeletePoolNotFound) GetPayload() interface{} {
	return o.Payload
}

func (o *DeletePoolNotFound) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewDeletePoolInternalServerError creates a DeletePoolInternalServerError with default headers values
func NewDeletePoolInternalServerError() *DeletePoolInternalServerError {
	return &DeletePoolInternalServerError{}
}

/* DeletePoolInternalServerError describes a response with status code 500, with default header values.

DeletePoolInternalServerError delete pool internal server error
*/
type DeletePoolInternalServerError struct {
	Payload interface{}
}

func (o *DeletePoolInternalServerError) Error() string {
	return fmt.Sprintf("[DELETE /pools/{pool_id}][%d] deletePoolInternalServerError  %+v", 500, o.Payload)
}
func (o *DeletePoolInternalServerError) GetPayload() interface{} {
	return o.Payload
}

func (o *DeletePoolInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
