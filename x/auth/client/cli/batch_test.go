package cli

import (
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"testing"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/spf13/viper"

	"github.com/cosmos/cosmos-sdk/tests"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/go-amino"
)

func TestGetBatchSignCommand(t *testing.T) {
	cdc := amino.NewCodec()
	cmd := GetBatchSignCommand(cdc)

	tempDir, cleanFunc := tests.NewTestCaseDir(t)
	t.Cleanup(cleanFunc)

	viper.Set(flags.FlagHome, tempDir)
	viper.Set(flags.FlagKeyringBackend, keyring.BackendTest)

	err := cmd.Execute()
	require.NoError(t, err)
}
