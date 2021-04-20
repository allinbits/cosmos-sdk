package mock

import (
	"encoding/json"
	"path/filepath"

	"github.com/tendermint/tendermint/types"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/libs/log"

	bam "github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewApp creates a simple mock kvstore app for testing. It should work
// similar to a real app. Make sure rootDir is empty before running the test,
// in order to guarantee consistent results
func NewApp(rootDir string, logger log.Logger) (abci.Application, error) {
	db, err := sdk.NewLevelDB("mock", filepath.Join(rootDir, "data"))
	if err != nil {
		return nil, err
	}

	// Capabilities key to access the main KVStore.
	capKeyMainStore := sdk.NewKVStoreKey("main")

	// Create BaseApp.
	baseApp := bam.NewBaseApp("kvstore", logger, db, decodeTx)

	// Set mounts for BaseApp's MultiStore.
	baseApp.MountStores(capKeyMainStore)

	baseApp.SetInitChainer(InitChainer(capKeyMainStore))

	// Load latest version.
	if err := baseApp.LoadLatestVersion(); err != nil {
		return nil, err
	}

	return baseApp, nil
}

// basic KV structure
type KV struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// What Genesis JSON is formatted as
type GenesisJSON struct {
	Values []KV `json:"values"`
}

// InitChainer returns a function that can initialize the chain
// with key/value pairs
func InitChainer(key sdk.StoreKey) func(sdk.Context, abci.RequestInitChain) abci.ResponseInitChain {
	return func(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
		stateJSON := req.AppStateBytes

		genesisState := new(GenesisJSON)
		err := json.Unmarshal(stateJSON, genesisState)
		if err != nil {
			panic(err) // TODO https://github.com/cosmos/cosmos-sdk/issues/468
			// return sdk.ErrGenesisParse("").TraceCause(err, "")
		}

		for _, val := range genesisState.Values {
			store := ctx.KVStore(key)
			store.Set([]byte(val.Key), []byte(val.Value))
		}
		return abci.ResponseInitChain{}
	}
}

// AppGenState can be passed into InitCmd, returns a static string of a few
// key-values that can be parsed by InitChainer
func AppGenState(_ *codec.LegacyAmino, _ types.GenesisDoc, _ []json.RawMessage) (appState json.
	RawMessage, err error) {
	appState = json.RawMessage(`{
  "values": [
    {
        "key": "hello",
        "value": "goodbye"
    },
    {
        "key": "foo",
        "value": "bar"
    }
  ]
}`)
	return
}

// AppGenStateEmpty returns an empty transaction state for mocking.
func AppGenStateEmpty(_ *codec.LegacyAmino, _ types.GenesisDoc, _ []json.RawMessage) (
	appState json.RawMessage, err error) {
	appState = json.RawMessage(``)
	return
}
