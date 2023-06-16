// Code generated by go-swagger; DO NOT EDIT.

package backend

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// UpdateHeartBeatHandlerFunc turns a function with the right signature into a update heart beat handler
type UpdateHeartBeatHandlerFunc func(UpdateHeartBeatParams) middleware.Responder

// Handle executing the request and returning a response
func (fn UpdateHeartBeatHandlerFunc) Handle(params UpdateHeartBeatParams) middleware.Responder {
	return fn(params)
}

// UpdateHeartBeatHandler interface for that can handle valid update heart beat params
type UpdateHeartBeatHandler interface {
	Handle(UpdateHeartBeatParams) middleware.Responder
}

// NewUpdateHeartBeat creates a new http.Handler for the update heart beat operation
func NewUpdateHeartBeat(ctx *middleware.Context, handler UpdateHeartBeatHandler) *UpdateHeartBeat {
	return &UpdateHeartBeat{Context: ctx, Handler: handler}
}

/*
	UpdateHeartBeat swagger:route PUT /namespaces/{namespace}/devices/{device-id}/heartbeat backend updateHeartBeat

Updates the heartbeat information of the device.
*/
type UpdateHeartBeat struct {
	Context *middleware.Context
	Handler UpdateHeartBeatHandler
}

func (o *UpdateHeartBeat) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewUpdateHeartBeatParams()
	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}
