package app

import (
	"io"
	"os"

	bam "github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/version"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/crisis"
	distr "github.com/cosmos/cosmos-sdk/x/distribution"
	"github.com/cosmos/cosmos-sdk/x/genaccounts"
	"github.com/cosmos/cosmos-sdk/x/genutil"
	"github.com/cosmos/cosmos-sdk/x/gov"
	"github.com/cosmos/cosmos-sdk/x/mint"
	"github.com/cosmos/cosmos-sdk/x/params"
	"github.com/cosmos/cosmos-sdk/x/slashing"
	"github.com/cosmos/cosmos-sdk/x/staking"

	abci "github.com/tendermint/tendermint/abci/types"
	cmn "github.com/tendermint/tendermint/libs/common"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
)

const appName = "GaiaApp"

var (
	// ensure GaiaApp fulfills the abci application interface
	_ abci.Application = GaiaApp{}

	// default home directories for gaiacli
	DefaultCLIHome = os.ExpandEnv("$HOME/.gaiacli")

	// default home directories for gaiad
	DefaultNodeHome = os.ExpandEnv("$HOME/.gaiad")

	// The ModuleBasicManager is in charge of setting up basic,
	// non-dependant module elements, such as codec registration
	// and genesis verification.
	ModuleBasics sdk.ModuleBasicManager
)

func init() {
	ModuleBasics = sdk.NewModuleBasicManager(
		genaccounts.AppModuleBasic{},
		genutil.AppModuleBasic{},
		auth.AppModuleBasic{},
		bank.AppModuleBasic{},
		staking.AppModuleBasic{},
		mint.AppModuleBasic{},
		distr.AppModuleBasic{},
		gov.AppModuleBasic{},
		params.AppModuleBasic{},
		crisis.AppModuleBasic{},
		slashing.AppModuleBasic{},
	)
}

// custom tx codec
func MakeCodec() *codec.Codec {
	var cdc = codec.New()
	ModuleBasics.RegisterCodec(cdc)
	sdk.RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)
	return cdc
}

// Extended ABCI application
type GaiaApp struct {
	*bam.BaseApp
	cdc *codec.Codec

	invCheckPeriod uint

	// keys to access the substores
	keyMain          *sdk.KVStoreKey
	keyAccount       *sdk.KVStoreKey
	tkeyStaking      *sdk.TransientStoreKey
	keyStaking       *sdk.KVStoreKey
	keySlashing      *sdk.KVStoreKey
	keyMint          *sdk.KVStoreKey
	keyDistr         *sdk.KVStoreKey
	tkeyDistr        *sdk.TransientStoreKey
	keyGov           *sdk.KVStoreKey
	keyFeeCollection *sdk.KVStoreKey
	keyParams        *sdk.KVStoreKey
	tkeyParams       *sdk.TransientStoreKey

	// keepers
	accountKeeper       auth.AccountKeeper
	bankKeeper          bank.Keeper
	crisisKeeper        crisis.Keeper
	distrKeeper         distr.Keeper
	feeCollectionKeeper auth.FeeCollectionKeeper
	govKeeper           gov.Keeper
	mintKeeper          mint.Keeper
	paramsKeeper        params.Keeper
	slashingKeeper      slashing.Keeper
	stakingKeeper       staking.Keeper

	// modules
	accountsMod sdk.AppModule
	genutilMod  sdk.AppModule
	authMod     sdk.AppModule
	bankMod     sdk.AppModule
	crisisMod   sdk.AppModule
	distrMod    sdk.AppModule
	govMod      sdk.AppModule
	mintMod     sdk.AppModule
	slashingMod sdk.AppModule
	stakingMod  sdk.AppModule
}

// the app modules
func (app *GaiaApp) modules() []sdk.Module {
	return []sdk.Module{app.accountsMod, app.genutilMod, app.authMod,
		app.bankMod, app.crisisMod, app.distrMod, app.govMod, app.mintMod,
		app.slashingMod, app.stakingMod,
	}
}

// the app keys
func (app *GaiaApp) keys() []sdk.StoreKey {
	return []sdk.StoreKey{keyMain, keyAccount, tkeyStaking, keyStaking,
		keySlashing, keyMint, keyDistr, tkeyDistr, keyGov, keyFeeCollection,
		keyParams, tkeyParams,
	}
}

// NewGaiaApp returns a reference to an initialized GaiaApp.
func NewGaiaApp(logger log.Logger, db dbm.DB, traceStore io.Writer, loadLatest bool,
	invCheckPeriod uint, baseAppOptions ...func(*bam.BaseApp)) *GaiaApp {

	cdc := MakeCodec()

	bApp := bam.NewBaseApp(appName, logger, db, auth.DefaultTxDecoder(cdc), baseAppOptions...)
	bApp.SetCommitMultiStoreTracer(traceStore)
	bApp.SetAppVersion(version.Version)

	var app = &GaiaApp{
		BaseApp:          bApp,
		cdc:              cdc,
		invCheckPeriod:   invCheckPeriod,
		keyMain:          sdk.NewKVStoreKey(bam.MainStoreKey),
		keyAccount:       sdk.NewKVStoreKey(auth.StoreKey),
		keyStaking:       sdk.NewKVStoreKey(staking.StoreKey),
		tkeyStaking:      sdk.NewTransientStoreKey(staking.TStoreKey),
		keyMint:          sdk.NewKVStoreKey(mint.StoreKey),
		keyDistr:         sdk.NewKVStoreKey(distr.StoreKey),
		tkeyDistr:        sdk.NewTransientStoreKey(distr.TStoreKey),
		keySlashing:      sdk.NewKVStoreKey(slashing.StoreKey),
		keyGov:           sdk.NewKVStoreKey(gov.StoreKey),
		keyFeeCollection: sdk.NewKVStoreKey(auth.FeeStoreKey),
		keyParams:        sdk.NewKVStoreKey(params.StoreKey),
		tkeyParams:       sdk.NewTransientStoreKey(params.TStoreKey),
	}

	// init params keeper and subspaces
	app.paramsKeeper = params.NewKeeper(app.cdc, app.keyParams, app.tkeyParams, params.DefaultCodespace)
	authSubspace := app.paramsKeeper.Subspace(auth.DefaultParamspace)
	bankSubspace := app.paramsKeeper.Subspace(bank.DefaultParamspace)
	stakingSubspace := app.paramsKeeper.Subspace(staking.DefaultParamspace)
	mintSubspace := app.paramsKeeper.Subspace(mint.DefaultParamspace)
	distrSubspace := app.paramsKeeper.Subspace(distr.DefaultParamspace)
	slashingSubspace := app.paramsKeeper.Subspace(slashing.DefaultParamspace)
	govSubspace := app.paramsKeeper.Subspace(gov.DefaultParamspace)
	crisisSubspace := app.paramsKeeper.Subspace(crisis.DefaultParamspace)

	// add keepers
	app.accountKeeper = auth.NewAccountKeeper(app.cdc, app.keyAccount, authSubspace, auth.ProtoBaseAccount)
	app.bankKeeper = bank.NewBaseKeeper(app.accountKeeper, bankSubspace, bank.DefaultCodespace)
	app.feeCollectionKeeper = auth.NewFeeCollectionKeeper(app.cdc, app.keyFeeCollection)
	stakingKeeper := staking.NewKeeper(app.cdc, app.keyStaking, app.tkeyStaking, app.bankKeeper,
		stakingSubspace, staking.DefaultCodespace)
	app.mintKeeper = mint.NewKeeper(app.cdc, app.keyMint, mintSubspace, &stakingKeeper, app.feeCollectionKeeper)
	app.distrKeeper = distr.NewKeeper(app.cdc, app.keyDistr, distrSubspace, app.bankKeeper, &stakingKeeper,
		app.feeCollectionKeeper, distr.DefaultCodespace)
	app.slashingKeeper = slashing.NewKeeper(app.cdc, app.keySlashing, &stakingKeeper,
		slashingSubspace, slashing.DefaultCodespace)
	app.crisisKeeper = crisis.NewKeeper(crisisSubspace, invCheckPeriod, app.distrKeeper,
		app.bankKeeper, app.feeCollectionKeeper)

	// register the proposal types
	govRouter := gov.NewRouter()
	govRouter.AddRoute(gov.RouterKey, gov.ProposalHandler).
		AddRoute(params.RouterKey, params.NewParamChangeProposalHandler(app.paramsKeeper))
	app.govKeeper = gov.NewKeeper(app.cdc, app.keyGov, app.paramsKeeper, govSubspace,
		app.bankKeeper, &stakingKeeper, gov.DefaultCodespace, govRouter)

	// register the staking hooks
	// NOTE: stakingKeeper above is passed by reference, so that it will contain these hooks
	app.stakingKeeper = *stakingKeeper.SetHooks(
		staking.NewMultiStakingHooks(app.distrKeeper.Hooks(), app.slashingKeeper.Hooks()))

	app.accountsMod = genaccounts.NewAppModule(app.accountKeeper)
	app.genutilMod = genutil.NewAppModule(app.accountKeeper, app.stakingKeeper, app.BaseApp.DeliverTx)
	app.authMod = auth.NewAppModule(app.accountKeeper, app.feeCollectionKeeper)
	app.bankMod = bank.NewAppModule(app.bankKeeper, app.accountKeeper)
	app.crisisMod = crisis.NewAppModule(app.crisisKeeper, app.Logger())
	app.distrMod = distr.NewAppModule(app.distrKeeper)
	app.govMod = gov.NewAppModule(app.govKeeper)
	app.mintMod = mint.NewAppModule(app.mintKeeper)
	app.slashingMod = slashing.NewAppModule(app.slashingKeeper, app.stakingKeeper)
	app.stakingMod = staking.NewAppModule(app.stakingKeeper, app.feeCollectionKeeper, app.distrKeeper, app.accountKeeper)

	sdk.RegisterInvariants(&app.crisisKeeper, app.modules())
	sdk.RegisterRoutes(app.Router(), app.QueryRouter(), app.modules())
	app.MountStores(app.keys())

	// initialize BaseApp
	app.SetInitChainer(app.InitChainer)
	app.SetBeginBlocker(app.BeginBlocker)
	app.SetAnteHandler(auth.NewAnteHandler(app.accountKeeper, app.feeCollectionKeeper, auth.DefaultSigVerificationGasConsumer))
	app.SetEndBlocker(app.EndBlocker)

	if loadLatest {
		err := app.LoadLatestVersion(app.keyMain)
		if err != nil {
			cmn.Exit(err.Error())
		}
	}
	return app
}

// application updates every begin block
func (app *GaiaApp) BeginBlocker(ctx sdk.Context, req abci.RequestBeginBlock) abci.ResponseBeginBlock {

	// During begin block slashing happens after distr.BeginBlocker so that
	// there is nothing left over in the validator fee pool, so as to keep the
	// CanWithdrawInvariant invariant.
	mintTags := app.mintMod.BeginBlock(ctx, req)
	distrTags := app.distrMod.BeginBlock(ctx, req)
	slashingTags := app.slashingMod.BeginBlock(ctx, req)
	tags := mintTags.AppendTags(distrTags).AppendTags(slashingTags)

	return abci.ResponseBeginBlock{
		Tags: tags,
	}
}

// application updates every end block
func (app *GaiaApp) EndBlocker(ctx sdk.Context, req abci.RequestEndBlock) abci.ResponseEndBlock {

	_, govTags := app.govMod.EndBlock(ctx, req)
	validatorUpdates, stakingTags := app.stakingMod.EndBlock(ctx, req)
	tags := govTags.AppendTags(stakingTags)

	return abci.ResponseEndBlock{
		ValidatorUpdates: validatorUpdates,
		Tags:             tags,
	}
}

// application update at chain initialization
func (app *GaiaApp) InitChainer(ctx sdk.Context, req abci.RequestInitChain) abci.ResponseInitChain {
	var genesisState GenesisState
	app.cdc.MustUnmarshalJSON(req.AppStateBytes, &genesisState)

	// genutils must occur after staking so that pools are properly
	// initialized with tokens from genesis accounts.
	app.accountsMod.InitGenesis(ctx, genesisState[app.accountsMod.Name()])
	app.distrMod.InitGenesis(ctx, genesisState[app.distrMod.Name()])
	app.stakingMod.InitGenesis(ctx, genesisState[app.stakingMod.Name()])
	app.authMod.InitGenesis(ctx, genesisState[app.authMod.Name()])
	app.bankMod.InitGenesis(ctx, genesisState[app.bankMod.Name()])
	app.slashingMod.InitGenesis(ctx, genesisState[app.slashingMod.Name()])
	app.govMod.InitGenesis(ctx, genesisState[app.govMod.Name()])
	app.mintMod.InitGenesis(ctx, genesisState[app.mintMod.Name()])
	app.crisisMod.InitGenesis(ctx, genesisState[app.crisisMod.Name()])
	app.genutilMod.InitGenesis(ctx, genesisState[app.genutilMod.Name()])

	return app.mm.InitGenesis(ctx, genesisState)
}

// load a particular height
func (app *GaiaApp) LoadHeight(height int64) error {
	return app.LoadVersion(height, app.keyMain)
}
