/*
Package types contains application module patterns and associated "manager"
functionality.  The module pattern has been broken down by:
 - independent module functionality (AppModuleBasic)
 - inter-dependent module functionality (AppModule)

inter-dependent module functionality is module functionality which somehow
depends on other modules, typically through the module keeper.  Many of the
module keepers are dependent on each other, thus in order to access the full
set of module functionality we need to define all the keepers/params-store/keys
etc. This full set of advanced functionality is defined by the AppModule interface.

Independent module functions are separated to allow for the construction of the
basic application structures required early on in the application definition
and used to enable the definition of full module functionality later in the
process. This separation is necessary, however we still want to allow for a
high level pattern for modules to follow - for instance, such that we don't
have to manually register all of the codecs for all the modules. This basic
procedure as well as other basic patterns are handled through the use of
ModuleBasicManager.
*/
package types

import (
	"encoding/json"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/spf13/cobra"
	abci "github.com/tendermint/tendermint/abci/types"
)

// ModuleClient helps modules provide a standard interface for exporting client functionality
type ModuleClient interface {
	GetQueryCmd() *cobra.Command
	GetTxCmd() *cobra.Command
}

//__________________________________________________________________________________________
// AppModule is the standard form for basic non-dependant elements of an application module.
type AppModuleBasic interface {
	Name() string
	RegisterCodec(*codec.Codec)
	DefaultGenesis() json.RawMessage
	ValidateGenesis(json.RawMessage) error
}

// collections of AppModuleBasic
type ModuleBasicManager []AppModuleBasic

func NewModuleBasicManager(modules ...AppModuleBasic) ModuleBasicManager {
	return modules
}

// RegisterCodecs registers all module codecs
func (mbm ModuleBasicManager) RegisterCodec(cdc *codec.Codec) {
	for _, mb := range mbm {
		mb.RegisterCodec(cdc)
	}
}

// Provided default genesis information for all modules
func (mbm ModuleBasicManager) DefaultGenesis() map[string]json.RawMessage {
	genesis := make(map[string]json.RawMessage)
	for _, mb := range mbm {
		genesis[mb.Name()] = mb.DefaultGenesis()
	}
	return genesis
}

// Provided default genesis information for all modules
func (mbm ModuleBasicManager) ValidateGenesis(genesis map[string]json.RawMessage) error {
	for _, mb := range mbm {
		if err := mb.ValidateGenesis(genesis[mb.Name()]); err != nil {
			return err
		}
	}
	return nil
}

//_________________________________________________________
// AppModule is the standard form for an application module
type AppModule interface {
	AppModuleBasic

	// registers
	RegisterInvariants(InvariantRouter)

	// routes
	Route() string
	NewHandler() Handler
	QuerierRoute() string
	NewQuerierHandler() Querier

	// genesis
	InitGenesis(Context, json.RawMessage) []abci.ValidatorUpdate
	ExportGenesis(Context) json.RawMessage

	BeginBlock(Context, abci.RequestBeginBlock) Tags
	EndBlock(Context, abci.RequestEndBlock) ([]abci.ValidatorUpdate, Tags)
}

// helper function to register all module invariants
func RegisterInvariants(invarRouter InvariantRouter, modules []AppModules) {
	for _, module := range Modules {
		module.RegisterInvariants(invarRouter)
	}
}

// helper function to register all module routes and module querier routes
func RegisterRoutesAndInvariants(router Router, queryRouter QueryRouter, modules []AppModules) {
	for _, module := range Modules {
		if module.Route() != "" {
			router.AddRoute(module.Route(), module.NewHandler())
		}
		if module.QuerierRoute() != "" {
			queryRouter.AddRoute(module.QuerierRoute(), module.NewQuerierHandler())
		}
	}
}

// DONTCOVER
