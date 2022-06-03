// Code generated by go-swagger; DO NOT EDIT.

package backend

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/strfmt"

	"github.com/project-flotta/flotta-operator/backend/models"
)

// RegisterDeviceReader is a Reader for the RegisterDevice structure.
type RegisterDeviceReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *RegisterDeviceReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {
	case 200:
		result := NewRegisterDeviceOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil
	case 401:
		result := NewRegisterDeviceUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	case 403:
		result := NewRegisterDeviceForbidden()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result
	default:
		result := NewRegisterDeviceDefault(response.Code())
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		if response.Code()/100 == 2 {
			return result, nil
		}
		return nil, result
	}
}

// NewRegisterDeviceOK creates a RegisterDeviceOK with default headers values
func NewRegisterDeviceOK() *RegisterDeviceOK {
	return &RegisterDeviceOK{}
}

/* RegisterDeviceOK describes a response with status code 200, with default header values.

Updated
*/
type RegisterDeviceOK struct {
}

func (o *RegisterDeviceOK) Error() string {
	return fmt.Sprintf("[PUT /namespaces/{namespace}/devices/{device-id}/registration][%d] registerDeviceOK ", 200)
}

func (o *RegisterDeviceOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewRegisterDeviceUnauthorized creates a RegisterDeviceUnauthorized with default headers values
func NewRegisterDeviceUnauthorized() *RegisterDeviceUnauthorized {
	return &RegisterDeviceUnauthorized{}
}

/* RegisterDeviceUnauthorized describes a response with status code 401, with default header values.

Unauthorized
*/
type RegisterDeviceUnauthorized struct {
}

func (o *RegisterDeviceUnauthorized) Error() string {
	return fmt.Sprintf("[PUT /namespaces/{namespace}/devices/{device-id}/registration][%d] registerDeviceUnauthorized ", 401)
}

func (o *RegisterDeviceUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewRegisterDeviceForbidden creates a RegisterDeviceForbidden with default headers values
func NewRegisterDeviceForbidden() *RegisterDeviceForbidden {
	return &RegisterDeviceForbidden{}
}

/* RegisterDeviceForbidden describes a response with status code 403, with default header values.

Forbidden
*/
type RegisterDeviceForbidden struct {
}

func (o *RegisterDeviceForbidden) Error() string {
	return fmt.Sprintf("[PUT /namespaces/{namespace}/devices/{device-id}/registration][%d] registerDeviceForbidden ", 403)
}

func (o *RegisterDeviceForbidden) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	return nil
}

// NewRegisterDeviceDefault creates a RegisterDeviceDefault with default headers values
func NewRegisterDeviceDefault(code int) *RegisterDeviceDefault {
	return &RegisterDeviceDefault{
		_statusCode: code,
	}
}

/* RegisterDeviceDefault describes a response with status code -1, with default header values.

Error
*/
type RegisterDeviceDefault struct {
	_statusCode int

	Payload *models.Error
}

// Code gets the status code for the register device default response
func (o *RegisterDeviceDefault) Code() int {
	return o._statusCode
}

func (o *RegisterDeviceDefault) Error() string {
	return fmt.Sprintf("[PUT /namespaces/{namespace}/devices/{device-id}/registration][%d] RegisterDevice default  %+v", o._statusCode, o.Payload)
}
func (o *RegisterDeviceDefault) GetPayload() *models.Error {
	return o.Payload
}

func (o *RegisterDeviceDefault) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.Error)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}