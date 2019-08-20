package baseapp

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type router struct {
	routes map[string]sdk.Handler
}

var _ sdk.Router = NewRouter()

// NewRouter returns a reference to a new router.
//
// TODO: Either make the function private or make return type (router) public.
func NewRouter() *router { // nolint: golint
	return &router{
		routes: make(map[string]sdk.Handler),
	}
}

// AddRoute adds a route path to the router with a given handler. The route must
// be alphanumeric.
// TODO: enforce routes alphanumeric with '/' delimiter
func (rtr *router) AddRoute(path string, h sdk.Handler) sdk.Router {
	for route := range rtr.routes {
		// no two routes can be prefixes of one another
		if strings.HasPrefix(route, path) || strings.HasPrefix(path, route) {
			panic(fmt.Sprintf("Cannot register two routes that are prefixes of one another: %s, %s", route, path))
		}
	}

	rtr.routes[path] = h
	return rtr
}

// Route returns a handler for a given route path.
//
// TODO: Handle expressive matches.
func (rtr *router) Route(path string) sdk.Handler {
	for route := range rtr.routes {
		if strings.HasPrefix(path, route) {
			return rtr.routes[route]
		}
	}
	return nil
}
