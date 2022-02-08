// Code generated by go-swagger; DO NOT EDIT.

package groups

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/timdrysdale/relay/pkg/bc/models"
)

// DeletePoolsByGroupIDReader is a Reader for the DeletePoolsByGroupID structure.
type DeletePoolsByGroupIDReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *DeletePoolsByGroupIDReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewDeletePoolsByGroupIDOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewDeletePoolsByGroupIDUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 404:
		result := NewDeletePoolsByGroupIDNotFound()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 500:
		result := NewDeletePoolsByGroupIDInternalServerError()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		return nil, runtime.NewAPIError("response status code does not match any response statuses defined for this endpoint in the swagger spec", response, response.Code())
	}
}

// NewDeletePoolsByGroupIDOK creates a DeletePoolsByGroupIDOK with default headers values
func NewDeletePoolsByGroupIDOK() *DeletePoolsByGroupIDOK {
	return &DeletePoolsByGroupIDOK{}
}

/* DeletePoolsByGroupIDOK describes a response with status code 200, with default header values.

DeletePoolsByGroupIDOK delete pools by group Id o k
*/
type DeletePoolsByGroupIDOK struct {
	Payload models.IDList
}

func (o *DeletePoolsByGroupIDOK) Error() string {
	return fmt.Sprintf("[DELETE /groups/{group_id}/pools][%d] deletePoolsByGroupIdOK  %+v", 200, o.Payload)
}
func (o *DeletePoolsByGroupIDOK) GetPayload() models.IDList {
	return o.Payload
}

func (o *DeletePoolsByGroupIDOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewDeletePoolsByGroupIDUnauthorized creates a DeletePoolsByGroupIDUnauthorized with default headers values
func NewDeletePoolsByGroupIDUnauthorized() *DeletePoolsByGroupIDUnauthorized {
	return &DeletePoolsByGroupIDUnauthorized{}
}

/* DeletePoolsByGroupIDUnauthorized describes a response with status code 401, with default header values.

Unauthorized
*/
type DeletePoolsByGroupIDUnauthorized struct {
	Payload interface{}
}

func (o *DeletePoolsByGroupIDUnauthorized) Error() string {
	return fmt.Sprintf("[DELETE /groups/{group_id}/pools][%d] deletePoolsByGroupIdUnauthorized  %+v", 401, o.Payload)
}
func (o *DeletePoolsByGroupIDUnauthorized) GetPayload() interface{} {
	return o.Payload
}

func (o *DeletePoolsByGroupIDUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewDeletePoolsByGroupIDNotFound creates a DeletePoolsByGroupIDNotFound with default headers values
func NewDeletePoolsByGroupIDNotFound() *DeletePoolsByGroupIDNotFound {
	return &DeletePoolsByGroupIDNotFound{}
}

/* DeletePoolsByGroupIDNotFound describes a response with status code 404, with default header values.

Not Found
*/
type DeletePoolsByGroupIDNotFound struct {
}

func (o *DeletePoolsByGroupIDNotFound) Error() string {
	return fmt.Sprintf("[DELETE /groups/{group_id}/pools][%d] deletePoolsByGroupIdNotFound ", 404)
}

func (o *DeletePoolsByGroupIDNotFound) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewDeletePoolsByGroupIDInternalServerError creates a DeletePoolsByGroupIDInternalServerError with default headers values
func NewDeletePoolsByGroupIDInternalServerError() *DeletePoolsByGroupIDInternalServerError {
	return &DeletePoolsByGroupIDInternalServerError{}
}

/* DeletePoolsByGroupIDInternalServerError describes a response with status code 500, with default header values.

DeletePoolsByGroupIDInternalServerError delete pools by group Id internal server error
*/
type DeletePoolsByGroupIDInternalServerError struct {
	Payload interface{}
}

func (o *DeletePoolsByGroupIDInternalServerError) Error() string {
	return fmt.Sprintf("[DELETE /groups/{group_id}/pools][%d] deletePoolsByGroupIdInternalServerError  %+v", 500, o.Payload)
}
func (o *DeletePoolsByGroupIDInternalServerError) GetPayload() interface{} {
	return o.Payload
}

func (o *DeletePoolsByGroupIDInternalServerError) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	// response payload
	if err := consumer.Consume(response.Body(), &o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
