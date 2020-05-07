package module

import (
	"encoding/json"

	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type StreamingGenesisModule interface {
	//DefaultGenesisStreaming(writer GenesisWriter)
	//ValidateGenesisStreaming(reader GenesisReader) error
	ReadGenesis(ctx sdk.Context, reader ObjectReader)
	AfterReadGenesis(ctx sdk.Context) []abci.ValidatorUpdate
	//WriteGenesis(ctx sdk.Context, writer GenesisWriter)
}

type ValueReader = func(ptr interface{}) (more bool, err error)

type GenesisValidator interface {
	ExpectNumber(name string, validator func(x json.Number) error)
	ExpectString(name string, validator func(x string) error)
	ExpectBool(name string, validator func(x bool) error)
	ExpectArray(name string, validator func(reader ArrayReader) error)
	ExpectObject(name string, validator func(reader ObjectReader) error)
}

type ObjectReader interface {
	ReadObject(ptr interface{}) error
	OnNumber(name string, reader func(x json.Number) error)
	OnString(name string, reader func(x string) error)
	OnBool(name string, reader func(x bool) error)
	OnArray(name string, reader func(reader ArrayReader) error)
	OnObject(name string, reader func(reader ObjectReader) error)
}

type ArrayReader interface {
	ReadValue(ptr interface{}) error
	More() bool
}

type ObjectWriter interface {
	WriteNumber(name string, x json.Number)
	WriteFloat64(name string, x float64)
	WriteString(name string, x string)
	WriteBool(name string, x string)
	Write(name string)
	StartArray(name string) ArrayWriter
	StartObject(name string) ObjectWriter
	FinishObject()
}

type ArrayWriter interface {
}
