// Code generated by go-swagger; DO NOT EDIT.

package azure

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	middleware "github.com/go-openapi/runtime/middleware"
)

// CreateAzureRegionalClusterHandlerFunc turns a function with the right signature into a create azure regional cluster handler
type CreateAzureRegionalClusterHandlerFunc func(CreateAzureRegionalClusterParams) middleware.Responder

// Handle executing the request and returning a response
func (fn CreateAzureRegionalClusterHandlerFunc) Handle(params CreateAzureRegionalClusterParams) middleware.Responder {
	return fn(params)
}

// CreateAzureRegionalClusterHandler interface for that can handle valid create azure regional cluster params
type CreateAzureRegionalClusterHandler interface {
	Handle(CreateAzureRegionalClusterParams) middleware.Responder
}

// NewCreateAzureRegionalCluster creates a new http.Handler for the create azure regional cluster operation
func NewCreateAzureRegionalCluster(ctx *middleware.Context, handler CreateAzureRegionalClusterHandler) *CreateAzureRegionalCluster {
	return &CreateAzureRegionalCluster{Context: ctx, Handler: handler}
}

/*
CreateAzureRegionalCluster swagger:route POST /api/providers/azure/create azure createAzureRegionalCluster

Create Azure regional cluster
*/
type CreateAzureRegionalCluster struct {
	Context *middleware.Context
	Handler CreateAzureRegionalClusterHandler
}

func (o *CreateAzureRegionalCluster) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		r = rCtx
	}
	var Params = NewCreateAzureRegionalClusterParams()

	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request

	o.Context.Respond(rw, r, route.Produces, route, res)

}