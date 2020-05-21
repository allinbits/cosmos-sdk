package cli_test

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto/multisig"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	keys2 "github.com/cosmos/cosmos-sdk/crypto/keys"
	"github.com/cosmos/cosmos-sdk/tests"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmos/go-bip39"
	"github.com/tendermint/tendermint/crypto"
)

const passphrase = "012345678"

func TestGetBatchSignCommand(t *testing.T) {
	cdc := amino.NewCodec()
	sdk.RegisterCodec(cdc)
	staking.RegisterCodec(cdc)

	cmd := cli.GetBatchSignCommand(cdc)

	tempDir, cleanFunc := tests.NewTestCaseDir(t)
	t.Cleanup(cleanFunc)

	kb, _, err := createKeybaseWithMultisigAccount(tempDir)
	require.NoError(t, err)

	multiInfo, err := kb.Get("multi")
	require.NoError(t, err)

	viper.Reset()
	viper.Set(flags.FlagHome, tempDir)
	viper.Set(flags.FlagFrom, "acc1")
	viper.Set(cli.FlagMultisig, multiInfo.GetAddress())
	viper.Set(cli.FlagPassPhrase, passphrase)

	cmd.SetArgs([]string{
		"./testdata/txs.json",
		filepath.Join(tempDir, "outputfile"),
	})

	err = cmd.Execute()
	require.NoError(t, err)
}

func createKeybaseWithMultisigAccount(dir string) (keys2.Keybase, []crypto.PubKey, error) {
	kb, err := keys.NewKeyBaseFromDir(dir)
	if err != nil {
		return nil, nil, err
	}

	var pubKeys []crypto.PubKey
	for i := 0; i < 4; i++ {
		entropySeed, err := bip39.NewEntropy(256)
		if err != nil {
			return nil, nil, err
		}

		mnemonic, err := bip39.NewMnemonic(entropySeed[:])
		if err != nil {
			return nil, nil, err
		}

		account, err := kb.CreateAccount(
			fmt.Sprintf("acc%d", i),
			mnemonic,
			"",
			passphrase,
			0,
			0,
		)
		if err != nil {
			return nil, nil, err
		}

		pubKeys = append(pubKeys, account.GetPubKey())
	}

	pk := multisig.NewPubKeyMultisigThreshold(2, pubKeys)
	if _, err := kb.CreateMulti("multi", pk); err != nil {
		return nil, nil, err
	}

	return kb, pubKeys, nil
}

func TestGetBatchSignCommand_Error(t *testing.T) {
	tests := []struct {
		name          string
		errorContains string
		keybasePrep   func() (cleanFunc func(), tempDir string)
		providedFlags map[string]interface{}
	}{
		{
			name:          "flag multisign not provided",
			errorContains: "only multisig signature is supported",
			keybasePrep: func() (func(), string) {
				tempDir, cleanFunc := tests.NewTestCaseDir(t)
				t.Cleanup(cleanFunc)

				kb, err := keys.NewKeyBaseFromDir(tempDir)
				require.NoError(t, err)

				_, _, err = kb.CreateMnemonic("acc1", keys2.English, "", keys2.Secp256k1)
				require.NoError(t, err)

				return cleanFunc, tempDir
			},
		},
		{
			name:          "not existing key",
			errorContains: "key not found: Key not-existing not found",
			keybasePrep: func() (func(), string) {
				tempDir, cleanFunc := tests.NewTestCaseDir(t)
				t.Cleanup(cleanFunc)

				return cleanFunc, tempDir
			},
			providedFlags: map[string]interface{}{
				cli.FlagMultisig: "cosmos1pf7m2k50lv0pc27wjz3452vu2xqs8yevxhv7w3",
				flags.FlagFrom:   "not-existing",
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			cdc := amino.NewCodec()
			cmd := cli.GetBatchSignCommand(cdc)

			cleanFunc, tempDir := tt.keybasePrep()
			defer cleanFunc()

			viper.Reset()
			viper.Set(flags.FlagHome, tempDir)

			for key, val := range tt.providedFlags {
				viper.Set(key, val)
			}

			cmd.SetArgs([]string{
				"./testdata/txs.json",
				filepath.Join(tempDir, "outputfile"),
			})

			err := cmd.Execute()
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.errorContains)
		})
	}
}
