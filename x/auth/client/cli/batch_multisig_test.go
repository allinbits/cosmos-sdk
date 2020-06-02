package cli_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/viper"

	"github.com/stretchr/testify/require"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/multisig"

	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/keys"
	keys2 "github.com/cosmos/cosmos-sdk/crypto/keys"
	"github.com/cosmos/cosmos-sdk/tests"
	"github.com/cosmos/cosmos-sdk/x/auth/client/cli"
)

func TestGetBatchMultisigCommand(t *testing.T) {
	cdc := getCodec()

	cmd := cli.GetBatchMultisigCommand(cdc)

	tempDir, cleanFunc := tests.NewTestCaseDir(t)
	t.Cleanup(cleanFunc)

	outputFile, err := os.Create(filepath.Join(tempDir, "the-output"))
	require.NoError(t, err)
	defer outputFile.Close()

	_, _, err = createKeybaseWith4AccountsAndMultisig(tempDir)
	require.NoError(t, err)

	cmd.SetArgs([]string{
		"./testdata/txs-multi-batch.json",
		"multi1",
		"./testdata/txs-multi-sig1.json",
		"./testdata/txs-multi-sig2.json",
	})

	viper.Reset()
	viper.Set(flags.FlagHome, tempDir)
	viper.Set(flags.FlagChainID, "testnet")
	viper.Set(flags.FlagTrustNode, true)
	viper.Set(flags.FlagAccountNumber, 1)
	viper.Set(flags.FlagSequence, 12)

	err = cmd.Execute()
	require.NoError(t, err)
}

func createKeybaseWith4AccountsAndMultisig(dir string) (keys2.Keybase, []crypto.PubKey, error) {
	kb, err := keys.NewKeyBaseFromDir(dir)
	if err != nil {
		return nil, nil, err
	}

	var pubKeys []crypto.PubKey

	ac, err := kb.CreateAccount(
		"key1",
		"champion obvious wedding submit wagon birth modify suffer virtual edit food hen fault loyal shuffle pattern crater knife enhance protect safe negative annual tower",
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
		"sausage syrup smart poet nut milk motor habit message risk abandon diagram wise mean anxiety submit shallow powder vapor merge thrive pen worry process",
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
		"key3",
		"genuine robust bounce turtle scatter water impact decade tribe combine symbol manual violin novel muffin basket coil bird panel honey meadow absurd tool bitter",
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
		"key4",
		"about under night giggle near boss tank valley lobster clump cushion broom ladder jeans lift ritual pattern winner basket track balcony eyebrow consider clown",
		"",
		passphrase,
		0,
		0,
	)
	if err != nil {
		return nil, nil, err
	}
	pubKeys = append(pubKeys, ac.GetPubKey())

	pk := multisig.NewPubKeyMultisigThreshold(2, pubKeys)
	if _, err := kb.CreateMulti("multi1", pk); err != nil {
		return nil, nil, err
	}

	return kb, pubKeys, nil
}
