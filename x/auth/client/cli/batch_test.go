package cli_test

import (
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
	"github.com/tendermint/tendermint/crypto"
)

const (
	passphrase    = "012345678"
	accountnumber = 123
	sequence      = 35
	chainId       = "the-chain-id"
)

func TestGetBatchSignCommand(t *testing.T) {
	cdc := getCodec()

	err := mockStdin(t, "./testdata/cmd-stdin")
	require.NoError(t, err)

	cmd := cli.GetBatchSignCommand(cdc)

	tempDir, cleanFunc := tests.NewTestCaseDir(t)
	t.Cleanup(cleanFunc)

	outputFile, err := os.Create(filepath.Join(tempDir, "the-output"))
	require.NoError(t, err)
	defer outputFile.Close()

	_, _, err = createKeybaseWithMultisigAccount(tempDir)
	require.NoError(t, err)

	viper.Reset()
	viper.Set(flags.FlagHome, tempDir)
	viper.Set(flags.FlagFrom, "key1")
	viper.Set(cli.FlagPassPhrase, passphrase)
	viper.Set(flags.FlagOutputDocument, outputFile.Name())
	viper.Set(flags.FlagAccountNumber, accountnumber)
	viper.Set(flags.FlagSequence, sequence)
	viper.Set(flags.FlagChainID, chainId)
	viper.Set(flags.FlagTrustNode, true)

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

func TestGetBatchSignCommand_Multisign(t *testing.T) {
	cdc := getCodec()

	err := mockStdin(t, "./testdata/cmd-stdin")
	require.NoError(t, err)

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
	viper.Set(flags.FlagFrom, "key1")
	viper.Set(cli.FlagPassPhrase, passphrase)
	viper.Set(flags.FlagOutputDocument, outputFile.Name())
	viper.Set(flags.FlagAccountNumber, accountnumber)
	viper.Set(flags.FlagSequence, sequence)
	viper.Set(flags.FlagChainID, chainId)
	viper.Set(flags.FlagTrustNode, true)
	viper.Set(cli.FlagMultisig, multiInfo.GetAddress())

	cmd.SetArgs([]string{
		"./testdata/txs-multi.json",
	})

	err = cmd.Execute()
	require.NoError(t, err)
}

func mockStdin(t *testing.T, inputFile string) error {
	inputStdin, err := os.Open(inputFile)
	require.NoError(t, err)
	os.Stdin = inputStdin
	return err
}

func getCodec() *amino.Codec {
	cdc := amino.NewCodec()
	sdk.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)
	staking.RegisterCodec(cdc)
	auth.RegisterCodec(cdc)

	return cdc
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

	seq := uint64(sequence)

	var parsedTxs []auth.StdSignMsg
	for _, txLine := range inputLines {
		if len(txLine) == 0 {
			break
		}

		var parsedTx auth.StdTx

		err := cdc.UnmarshalJSON([]byte(txLine), &parsedTx)
		if err != nil {
			t.Errorf("error extracting tx: %s", err)
		}

		stdSignMsg := auth.StdSignMsg{
			ChainID:       chainId,
			AccountNumber: accountnumber,
			Sequence:      seq,
			Fee:           parsedTx.Fee,
			Msgs:          parsedTx.GetMsgs(),
			Memo:          parsedTx.GetMemo(),
		}

		parsedTxs = append(parsedTxs, stdSignMsg)

		seq++
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

	ac, err := kb.CreateAccount(
		"key1",
		"orbit juice speak next refuse ten capable release inherit tuna spawn inherit topple shoot rebuild merit door deal salute wire traffic want oxygen sustain",
		"",
		passphrase,
		0,
		0,
	)
	if err != nil {
		return nil, nil, err
	}
	pubKeys = append(pubKeys, ac.GetPubKey())

	ac, err = kb.CreateAccount(
		"key2",
		"sniff spoon rhythm affair begin crime love wrong elbow bus alien borrow fit buddy fatal anger elevator track toe will magic deputy patient camera",
		"",
		passphrase,
		0,
		0,
	)
	if err != nil {
		return nil, nil, err
	}
	pubKeys = append(pubKeys, ac.GetPubKey())

	pk := multisig.NewPubKeyMultisigThreshold(1, pubKeys)
	if _, err := kb.CreateMulti("multi", pk); err != nil {
		return nil, nil, err
	}

	return kb, pubKeys, nil
}

func TestGetBatchSignCommand_Error(t *testing.T) {
	tts := []struct {
		name           string
		errorContains  string
		keybasePrep    func(tempDir string)
		providedFlags  map[string]interface{}
		stdinFileInput string
	}{
		{
			name:          "invalid signing account",
			errorContains: "tx intended signer does not match the given signer: key2",
			keybasePrep: func(tempDir string) {
				createKeybaseWithMultisigAccount(tempDir)
			},
			providedFlags: map[string]interface{}{
				flags.FlagFrom:          "key2", // Expects key1
				cli.FlagPassPhrase:      passphrase,
				flags.FlagAccountNumber: 50,
				flags.FlagSequence:      50,
				flags.FlagChainID:       chainId,
				flags.FlagTrustNode:     true,
			},
			stdinFileInput: "./testdata/cmd-stdin",
		},
		{
			name:          "bad passphrase",
			errorContains: "invalid account password",
			keybasePrep: func(tempDir string) {
				createKeybaseWithMultisigAccount(tempDir)
			},
			providedFlags: map[string]interface{}{
				flags.FlagFrom:          "key1", // Expects key1
				cli.FlagPassPhrase:      passphrase,
				flags.FlagAccountNumber: 50,
				flags.FlagSequence:      50,
				flags.FlagChainID:       chainId,
				flags.FlagTrustNode:     true,
			},
			stdinFileInput: "./testdata/cmd-stdin-bad-passphrase",
		},
	}

	cdc := getCodec()

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

			err := mockStdin(t, tt.stdinFileInput)

			err = cmd.Execute()
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.errorContains)
		})
	}
}
