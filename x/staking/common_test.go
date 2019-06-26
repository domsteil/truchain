package staking

import (
	trubank "github.com/TruStory/truchain/x/bank"
	"github.com/TruStory/truchain/x/claim"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/store"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/bank"
	"github.com/cosmos/cosmos-sdk/x/params"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"
	dbm "github.com/tendermint/tendermint/libs/db"
	"github.com/tendermint/tendermint/libs/log"
)

type mockAuth struct {
	jailStatus   map[string]bool
	forceFailure bool
}

func newAuth() *mockAuth {
	return &mockAuth{
		jailStatus: make(map[string]bool),
	}
}

func (m *mockAuth) jail(address sdk.AccAddress) {
	m.jailStatus[address.String()] = true
}

func (m *mockAuth) fail() {
	m.forceFailure = true
}
func (m *mockAuth) IsJailed(ctx sdk.Context, address sdk.AccAddress) (bool, sdk.Error) {
	if m.forceFailure {
		m.forceFailure = false
		return false, sdk.ErrInternal("error")
	}
	j, _ := m.jailStatus[address.String()]
	if j {
		return true, nil
	}
	return false, nil
}

func (m *mockAuth) UnJail(ctx sdk.Context, address sdk.AccAddress) sdk.Error {
	if m.forceFailure {
		m.forceFailure = false
		return sdk.ErrInternal("error")
	}
	m.jailStatus[address.String()] = false
	return nil
}

type mockClaimKeeper struct {
}

func (mockClaimKeeper) Claim(ctx sdk.Context, id uint64) (claim.Claim, bool) {
	return claim.Claim{}, true
}

func mockDB() (sdk.Context, Keeper, auth.AccountKeeper, AuthKeeper) {
	db := dbm.NewMemDB()
	storeKey := sdk.NewKVStoreKey(ModuleName)
	accKey := sdk.NewKVStoreKey(auth.StoreKey)
	paramsKey := sdk.NewKVStoreKey(params.StoreKey)
	transientParamsKey := sdk.NewTransientStoreKey(params.TStoreKey)
	bankKey := sdk.NewKVStoreKey("bank")
	claimKey := sdk.NewKVStoreKey(claim.StoreKey)

	ms := store.NewCommitMultiStore(db)
	ms.MountStoreWithDB(accKey, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(storeKey, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(paramsKey, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(bankKey, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(claimKey, sdk.StoreTypeIAVL, db)
	ms.MountStoreWithDB(transientParamsKey, sdk.StoreTypeTransient, db)
	ms.LoadLatestVersion()

	ctx := sdk.NewContext(ms, abci.Header{}, false, log.NewNopLogger())

	// codec registration
	cdc := codec.New()
	auth.RegisterCodec(cdc)
	RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)

	// Keepers
	pk := params.NewKeeper(cdc, paramsKey, transientParamsKey, params.DefaultCodespace)
	accKeeper := auth.NewAccountKeeper(cdc, accKey, pk.Subspace(auth.DefaultParamspace), auth.ProtoBaseAccount)

	bankKeeper := bank.NewBaseKeeper(accKeeper,
		pk.Subspace(bank.DefaultParamspace),
		bank.DefaultCodespace,
	)

	trubankKeeper := trubank.NewKeeper(cdc, bankKey, bankKeeper, pk.Subspace(trubank.DefaultParamspace), trubank.DefaultCodespace)

	mockedAuth := newAuth()
	keeper := NewKeeper(cdc, storeKey, mockedAuth, trubankKeeper, mockClaimKeeper{}, pk.Subspace(DefaultParamspace), DefaultCodespace)
	InitGenesis(ctx, keeper, DefaultGenesisState())
	return ctx, keeper, accKeeper, mockedAuth
}

func createFakeFundedAccount(ctx sdk.Context, am auth.AccountKeeper, coins sdk.Coins) sdk.AccAddress {
	_, _, addr := keyPubAddr()
	baseAcct := auth.NewBaseAccountWithAddress(addr)
	_ = baseAcct.SetCoins(coins)
	am.SetAccount(ctx, &baseAcct)

	return addr
}

func keyPubAddr() (crypto.PrivKey, crypto.PubKey, sdk.AccAddress) {
	key := ed25519.GenPrivKey()
	pub := key.PubKey()
	addr := sdk.AccAddress(pub.Address())
	return key, pub, addr
}
