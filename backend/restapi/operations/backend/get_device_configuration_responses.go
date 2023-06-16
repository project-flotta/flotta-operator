// Code generated by go-swagger; DO NOT EDIT.

package backend

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"

	"github.com/project-flotta/flotta-operator/backend/models"
)

// GetDeviceConfigurationOKCode is the HTTP code returned for type GetDeviceConfigurationOK
const GetDeviceConfigurationOKCode int = 200

/*
GetDeviceConfigurationOK Success

swagger:response getDeviceConfigurationOK
*/
type GetDeviceConfigurationOK struct {

	/*
	  In: Body
	*/
	Payload *models.DeviceConfigurationResponse `json:"body,omitempty"`
}

// NewGetDeviceConfigurationOK creates GetDeviceConfigurationOK with default headers values
func NewGetDeviceConfigurationOK() *GetDeviceConfigurationOK {

	return &GetDeviceConfigurationOK{}
}

// WithPayload adds the payload to the get device configuration o k response
func (o *GetDeviceConfigurationOK) WithPayload(payload *models.DeviceConfigurationResponse) *GetDeviceConfigurationOK {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get device configuration o k response
func (o *GetDeviceConfigurationOK) SetPayload(payload *models.DeviceConfigurationResponse) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetDeviceConfigurationOK) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(200)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

// GetDeviceConfigurationUnauthorizedCode is the HTTP code returned for type GetDeviceConfigurationUnauthorized
const GetDeviceConfigurationUnauthorizedCode int = 401

/*
GetDeviceConfigurationUnauthorized Unauthorized

swagger:response getDeviceConfigurationUnauthorized
*/
type GetDeviceConfigurationUnauthorized struct {
}

// NewGetDeviceConfigurationUnauthorized creates GetDeviceConfigurationUnauthorized with default headers values
func NewGetDeviceConfigurationUnauthorized() *GetDeviceConfigurationUnauthorized {

	return &GetDeviceConfigurationUnauthorized{}
}

// WriteResponse to the client
func (o *GetDeviceConfigurationUnauthorized) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.Header().Del(runtime.HeaderContentType) //Remove Content-Type on empty responses

	rw.WriteHeader(401)
}

// GetDeviceConfigurationForbiddenCode is the HTTP code returned for type GetDeviceConfigurationForbidden
const GetDeviceConfigurationForbiddenCode int = 403

/*
GetDeviceConfigurationForbidden Forbidden

swagger:response getDeviceConfigurationForbidden
*/
type GetDeviceConfigurationForbidden struct {
}

// NewGetDeviceConfigurationForbidden creates GetDeviceConfigurationForbidden with default headers values
func NewGetDeviceConfigurationForbidden() *GetDeviceConfigurationForbidden {

	return &GetDeviceConfigurationForbidden{}
}

// WriteResponse to the client
func (o *GetDeviceConfigurationForbidden) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.Header().Del(runtime.HeaderContentType) //Remove Content-Type on empty responses

	rw.WriteHeader(403)
}

/*
GetDeviceConfigurationDefault Error

swagger:response getDeviceConfigurationDefault
*/
type GetDeviceConfigurationDefault struct {
	_statusCode int

	/*
	  In: Body
	*/
	Payload *models.DeviceConfigurationResponse `json:"body,omitempty"`
}

// NewGetDeviceConfigurationDefault creates GetDeviceConfigurationDefault with default headers values
func NewGetDeviceConfigurationDefault(code int) *GetDeviceConfigurationDefault {
	if code <= 0 {
		code = 500
	}

	return &GetDeviceConfigurationDefault{
		_statusCode: code,
	}
}

// WithStatusCode adds the status to the get device configuration default response
func (o *GetDeviceConfigurationDefault) WithStatusCode(code int) *GetDeviceConfigurationDefault {
	o._statusCode = code
	return o
}

// SetStatusCode sets the status to the get device configuration default response
func (o *GetDeviceConfigurationDefault) SetStatusCode(code int) {
	o._statusCode = code
}

// WithPayload adds the payload to the get device configuration default response
func (o *GetDeviceConfigurationDefault) WithPayload(payload *models.DeviceConfigurationResponse) *GetDeviceConfigurationDefault {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the get device configuration default response
func (o *GetDeviceConfigurationDefault) SetPayload(payload *models.DeviceConfigurationResponse) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *GetDeviceConfigurationDefault) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(o._statusCode)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}
