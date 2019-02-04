package vote

import (
	"encoding/json"
	"testing"

	params "github.com/TruStory/truchain/parameters"
	"github.com/TruStory/truchain/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
)

func TestCreateVoteMsg(t *testing.T) {
	ctx, k, ck := mockDB()

	h := NewHandler(k)
	assert.NotNil(t, h)

	storyID := createFakeStory(ctx, k.storyKeeper, ck)
	amount := sdk.NewCoin(params.StakeDenom, sdk.NewInt(15000000000))
	creator := sdk.AccAddress([]byte{1, 2})

	// give user some funds
	k.bankKeeper.AddCoins(ctx, creator, sdk.Coins{amount.Plus(amount)})

	argument := "test argument"
	_, err := k.challengeKeeper.Create(ctx, storyID, amount, argument, creator)
	assert.Nil(t, err)

	msg := NewCreateVoteMsg(storyID, amount, "valid comment", creator, true)
	assert.NotNil(t, msg)

	res := h(ctx, msg)
	idres := new(types.IDResult)
	_ = json.Unmarshal(res.Data, &idres)

	assert.Equal(t, int64(1), idres.ID)
}
