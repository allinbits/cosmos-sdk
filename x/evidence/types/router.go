package types

import (
	"fmt"
	"regexp"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/evidence/exported"
)

// isAlphaNumeric defines a regular expression for matching against alpha-numeric
// values.
var isAlphaNumeric = regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString

type (
	// Handler defines an agnostic Evidence handler. The handler is responsible
	// for executing all corresponding business logic necessary for verifying the
	// evidence as valid. In addition, the Handler may execute any necessary
	// slashing and potential jailing.
	Handler func(sdk.Context, exported.Evidence) error

	// Router defines a contract for which any Evidence handling module must
	// implement in order to route Evidence to registered Handlers.
	Router interface {
		AddRoute(r string, h Handler) Router
		HasRoute(r string) bool
		GetRoute(path string) Handler
		Seal()
		Sealed() bool
	}

	router struct {
		routes map[string]Handler
		sealed bool
	}
)

func NewRouter() Router {
	return &router{
		routes: make(map[string]Handler),
	}
}

// Seal prevents the router from any subsequent route handlers to be registered.
// Seal will panic if called more than once.
func (rtr *router) Seal() {
	if rtr.sealed {
		panic("router already sealed")
	}
	rtr.sealed = true
}

// Sealed returns a boolean signifying if the Router is sealed or not.
func (rtr router) Sealed() bool {
	return rtr.sealed
}

// AddRoute adds a governance handler for a given path. It returns the Router
// so AddRoute calls can be linked. It will panic if the router is sealed.
func (rtr *router) AddRoute(path string, h Handler) Router {
	if rtr.sealed {
		panic(fmt.Sprintf("router sealed; cannot register %s route handler", path))
	}
	if !isAlphaNumeric(path) {
		panic("route expressions can only contain alphanumeric characters")
	}
	if rtr.HasRoute(path) {
		panic(fmt.Sprintf("route %s has already been registered", path))
	}

	rtr.routes[path] = h
	return rtr
}

// HasRoute returns true if the router has a path registered or false otherwise.
func (rtr *router) HasRoute(path string) bool {
	return rtr.routes[path] != nil
}

// GetRoute returns a Handler for a given path.
func (rtr *router) GetRoute(path string) Handler {
	if !rtr.HasRoute(path) {
		panic(fmt.Sprintf("route does not exist for path %s", path))
	}
	return rtr.routes[path]
}
