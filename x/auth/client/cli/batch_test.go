package cli_test

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
	"github.com/tendermint/go-amino"
	"github.com/tendermint/tendermint/crypto/multisig"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	"github.com/cosmos/cosmos-sdk/codec"
	keys2 "github.com/cosmos/cosmos-sdk/crypto/keys"
	"github.com/cosmos/cosmos-sdk/tests"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/auth/client/cli"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmos/go-bip39"
	"github.com/tendermint/tendermint/crypto"
)

const passphrase = "012345678"

func TestGetBatchSignCommand(t *testing.T) {
	cdc := amino.NewCodec()
	sdk.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)
	staking.RegisterCodec(cdc)

	cmd := cli.GetBatchSignCommand(cdc)

	tempDir, cleanFunc := tests.NewTestCaseDir(t)
	t.Cleanup(cleanFunc)

	outputFile, err := os.Create(filepath.Join(tempDir, "the-output"))
	require.NoError(t, err)
	defer outputFile.Close()

	kb, _, err := createKeybaseWithMultisigAccount(tempDir)
	require.NoError(t, err)

	multiInfo, err := kb.Get("multi")
	require.NoError(t, err)

	viper.Reset()
	viper.Set(flags.FlagHome, tempDir)
	viper.Set(flags.FlagFrom, "acc1")
	viper.Set(cli.FlagMultisig, multiInfo.GetAddress())
	viper.Set(cli.FlagPassPhrase, passphrase)
	viper.Set(flags.FlagOutputDocument, outputFile.Name())

	cmd.SetArgs([]string{
		"./testdata/txs.json",
	})

	err = cmd.Execute()
	require.NoError(t, err)

	// Validate Result
	inputFile, err := os.Open("./testdata/txs.json")
	require.NoError(t, err)

	validateSignatures(t, cdc, inputFile, outputFile)
}

func validateSignatures(t *testing.T, cdc *codec.Codec, inputFile io.Reader, outputFile io.Reader) {
	inputData, err := ioutil.ReadAll(inputFile)
	require.NoError(t, err)

	outputData, err := ioutil.ReadAll(outputFile)
	require.NoError(t, err)

	txs := extractTxs(t, cdc, inputData)
	signatures := extractSignatures(t, cdc, outputData)

	if len(txs) != len(signatures) {
		t.Errorf("must be same amount of txs and signatures: '%d' txs, '%d' signatures", len(txs), len(signatures))
	}

	for i := 0; i < len(txs); i++ {
		require.True(t, signatures[i].PubKey.VerifyBytes(txs[i].Bytes(), signatures[i].Signature))
	}
}

func extractTxs(t *testing.T, cdc *codec.Codec, inputData []byte) []auth.StdSignMsg {
	inputLines := strings.Split(string(inputData), "\n")

	var parsedTxs []auth.StdSignMsg
	for _, txLine := range inputLines {
		if len(txLine) == 0 {
			break
		}

		var parsedTx auth.StdSignMsg

		err := cdc.UnmarshalJSON([]byte(txLine), &parsedTx)
		if err != nil {
			t.Errorf("error extracting tx: %s", err)
		}

		parsedTxs = append(parsedTxs, parsedTx)
	}

	return parsedTxs
}

func extractSignatures(t *testing.T, cdc *codec.Codec, outputData []byte) []auth.StdSignature {
	outputLines := strings.Split(string(outputData), "\n")

	var parsedSigs []auth.StdSignature
	for _, sigLine := range outputLines {
		if len(sigLine) == 0 {
			break
		}

		var parsedSig auth.StdSignature

		err := cdc.UnmarshalJSON([]byte(sigLine), &parsedSig)
		if err != nil {
			t.Errorf("error extracting tx: %s", err)
		}

		parsedSigs = append(parsedSigs, parsedSig)
	}

	return parsedSigs
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
	tts := []struct {
		name          string
		errorContains string
		keybasePrep   func(tempDir string)
		providedFlags map[string]interface{}
	}{
		{
			name:          "flag multisign not provided",
			errorContains: "only multisig signature is supported",
			keybasePrep: func(tempDir string) {
				kb, err := keys.NewKeyBaseFromDir(tempDir)
				require.NoError(t, err)

				_, _, err = kb.CreateMnemonic("acc1", keys2.English, "", keys2.Secp256k1)
				require.NoError(t, err)
			},
		},
		{
			name:          "not existing key",
			errorContains: "key not found: Key not-existing not found",
			keybasePrep: func(tempDir string) {
			},
			providedFlags: map[string]interface{}{
				cli.FlagMultisig: "cosmos1pf7m2k50lv0pc27wjz3452vu2xqs8yevxhv7w3",
				flags.FlagFrom:   "not-existing",
			},
		},
		{
			name:          "invalid passphrase",
			errorContains: "invalid account password",
			keybasePrep: func(tempDir string) {
				createKeybaseWithMultisigAccount(tempDir)
			},
			providedFlags: map[string]interface{}{
				cli.FlagMultisig: "cosmos1pf7m2k50lv0pc27wjz3452vu2xqs8yevxhv7w3",
				flags.FlagFrom:   "acc1",
			},
		},
	}

	cdc := amino.NewCodec()
	sdk.RegisterCodec(cdc)
	staking.RegisterCodec(cdc)

	for _, tt := range tts {
		tt := tt
		tempDir, cleanFunc := tests.NewTestCaseDir(t)

		t.Run(tt.name, func(t *testing.T) {
			defer cleanFunc()

			cmd := cli.GetBatchSignCommand(cdc)

			tt.keybasePrep(tempDir)

			viper.Reset()
			viper.Set(flags.FlagHome, tempDir)

			for key, val := range tt.providedFlags {
				viper.Set(key, val)
			}

			cmd.SetArgs([]string{
				"./testdata/txs.json",
			})

			err := cmd.Execute()
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.errorContains)
		})
	}
}
