package cli

import (
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/go-amino"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/tests"
)

func TestGetBatchSignCommand(t *testing.T) {
	cdc := amino.NewCodec()
	cmd := GetBatchSignCommand(cdc)

	tempDir, cleanFunc := tests.NewTestCaseDir(t)
	t.Cleanup(cleanFunc)

	viper.Set(flags.FlagHome, tempDir)

	cmd.SetArgs([]string{
		"./testdata/txs.json",
		filepath.Join(tempDir, "outputfile"),
	})

	err := cmd.Execute()
	require.NoError(t, err)
}
