package shared

import (
	"time"

	"github.com/cosmos/cosmos-sdk/x/params"
)

// constants
const (
	ModuleName        = "shared"
	DefaultParamspace = ModuleName
)

// ParamKeyTable for slashing module
func ParamKeyTable() params.KeyTable {
	return params.NewKeyTable().RegisterParamSet(&Params{})
}

// nolint - Keys for parameter access
var (
	KeyUnbondingTime = []byte("UnbondingTime")
	KeyMaxValidators = []byte("MaxValidators")
	KeyBondDenom     = []byte("BondDenom")
)

var _ params.ParamSet = (*Params)(nil)

// Params defines the high level settings for staking
type Params struct {
	UnbondingTime time.Duration `json:"unbonding_time" yaml:"unbonding_time"` // time duration of unbonding
	MaxValidators uint16        `json:"max_validators" yaml:"max_validators"` // maximum number of validators (max uint16 = 65535)
	BondDenom     string        `json:"bond_denom" yaml:"bond_denom"`         // bondable coin denomination
}
