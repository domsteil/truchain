package staking

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"

	app "github.com/TruStory/truchain/types"
)

func TestKeeper_SubmitArgument(t *testing.T) {
	ctx, k, accKeeper, authKeeper := mockDB()
	ctx.WithBlockTime(time.Now())
	mockedAuth := authKeeper.(*mockAuth)
	addr := createFakeFundedAccount(ctx, accKeeper, sdk.Coins{sdk.NewInt64Coin(app.StakeDenom, app.Shanev*300)})
	addr2 := createFakeFundedAccount(ctx, accKeeper, sdk.Coins{sdk.NewInt64Coin(app.StakeDenom, app.Shanev*300)})
	mockedAuth.jail(addr)

	_, err := k.SubmitArgument(ctx, "body", "summary", addr, 1, StakeUpvote)
	assert.Error(t, err)
	assert.Equal(t, ErrorCodeInvalidStakeType, err.Code())

	_, err = k.SubmitArgument(ctx, "body", "summary", addr, 1, StakeType(0xFF))
	assert.Error(t, err)
	assert.Equal(t, ErrorCodeInvalidStakeType, err.Code())

	_, err = k.SubmitArgument(ctx, "body", "summary", addr, 1, StakeBacking)
	assert.Error(t, err)
	assert.Equal(t, ErrorCodeAccountJailed, err.Code())
	authKeeper.UnJail(ctx, addr)

	argument, err := k.SubmitArgument(ctx, "body", "summary", addr, 1, StakeBacking)
	assert.NoError(t, err)
	expectedArgument := Argument{
		ID:           1,
		Creator:      addr,
		ClaimID:      1,
		Summary:      "summary",
		Body:         "body",
		StakeType:    StakeBacking,
		CreatedTime:  ctx.BlockHeader().Time,
		UpdatedTime:  ctx.BlockHeader().Time,
		UpvotedCount: 1,
		UpvotedStake: sdk.NewInt64Coin(app.StakeDenom, 50),
	}
	assert.Equal(t, expectedArgument, argument)
	argument, ok := k.getArgument(ctx, expectedArgument.ID)
	assert.True(t, ok)
	assert.Equal(t, expectedArgument, argument)

	expectedStake := Stake{
		ID:          1,
		ArgumentID:  1,
		Type:        StakeBacking,
		Amount:      sdk.NewInt64Coin(app.StakeDenom, 50),
		Creator:     addr,
		CreatedTime: ctx.BlockHeader().Time,
		EndTime:     ctx.BlockHeader().Time.Add(time.Hour * 24 * 7),
	}
	assert.Equal(t, expectedStake, k.getStake(ctx, 1))
	argument2, err := k.SubmitArgument(ctx, "body2", "summary2", addr2, 1, StakeChallenge)
	expectedArgument2 := Argument{
		ID:           2,
		Creator:      addr2,
		ClaimID:      1,
		Summary:      "summary2",
		Body:         "body2",
		StakeType:    StakeChallenge,
		CreatedTime:  ctx.BlockHeader().Time,
		UpdatedTime:  ctx.BlockHeader().Time,
		UpvotedCount: 1,
		UpvotedStake: sdk.NewInt64Coin(app.StakeDenom, 50),
	}
	expectedStake2 := Stake{
		ID:          2,
		ArgumentID:  2,
		Type:        StakeChallenge,
		Amount:      sdk.NewInt64Coin(app.StakeDenom, 50),
		Creator:     addr2,
		CreatedTime: ctx.BlockHeader().Time,
		EndTime:     ctx.BlockHeader().Time.Add(time.Hour * 24 * 7),
	}
	assert.NoError(t, err)
	assert.Equal(t, expectedArgument2, argument2)
	assert.Equal(t, expectedStake2, k.getStake(ctx, 2))
	associatedArguments := k.ClaimArguments(ctx, 1)
	assert.Len(t, associatedArguments, 2)
	assert.Equal(t, expectedArgument, associatedArguments[0])
	assert.Equal(t, expectedArgument2, associatedArguments[1])

	associatedStakes := k.ArgumentStakes(ctx, expectedArgument.ID)
	assert.Len(t, associatedStakes, 1)
	assert.Equal(t, associatedStakes[0], expectedStake)

	// user <-> argument associations
	user1Arguments := k.UserArguments(ctx, addr)
	user2Arguments := k.UserArguments(ctx, addr2)

	assert.Len(t, user1Arguments, 1)
	assert.Len(t, user2Arguments, 1)

	assert.Equal(t, user1Arguments[0], expectedArgument)
	assert.Equal(t, user2Arguments[0], expectedArgument2)

	// user <-> stakes

	user1Stakes := k.UserStakes(ctx, addr)
	user2Stakes := k.UserStakes(ctx, addr2)

	assert.Len(t, user1Stakes, 1)
	assert.Len(t, user2Stakes, 1)

	assert.Equal(t, user1Stakes[0], expectedStake)
	assert.Equal(t, user2Stakes[0], expectedStake2)

	expiringStakes := make([]Stake, 0)

	k.IterateActiveStakeQueue(ctx, ctx.BlockHeader().Time, func(stake Stake) bool {
		expiringStakes = append(expiringStakes, stake)
		return false
	})
	// shouldn't have any expiring stake
	assert.Len(t, expiringStakes, 0)

	period := k.GetParams(ctx).Period
	k.IterateActiveStakeQueue(ctx, ctx.BlockHeader().Time.Add(period), func(stake Stake) bool {
		expiringStakes = append(expiringStakes, stake)
		return false
	})

	assert.Len(t, expiringStakes, 2)
	assert.Equal(t, []Stake{expectedStake, expectedStake2}, expiringStakes)
}

func TestKeeper_SubmitUpvote(t *testing.T) {
	ctx, k, accKeeper, _ := mockDB()
	ctx.WithBlockTime(time.Now())
	addr := createFakeFundedAccount(ctx, accKeeper, sdk.Coins{sdk.NewInt64Coin(app.StakeDenom, app.Shanev*300)})
	addr2 := createFakeFundedAccount(ctx, accKeeper, sdk.Coins{sdk.NewInt64Coin(app.StakeDenom, app.Shanev*300)})
	argument, err := k.SubmitArgument(ctx, "body", "summary", addr, 1, StakeBacking)
	assert.NoError(t, err)
	expectedStake := Stake{
		ID:          1,
		ArgumentID:  1,
		Type:        StakeBacking,
		Amount:      sdk.NewInt64Coin(app.StakeDenom, 50),
		Creator:     addr,
		CreatedTime: ctx.BlockHeader().Time,
		EndTime:     ctx.BlockHeader().Time.Add(time.Hour * 24 * 7),
	}
	assert.Equal(t, expectedStake, k.getStake(ctx, 1))
	_, err = k.SubmitUpvote(ctx, argument.ID, addr2)
	assert.NoError(t, err)
	expectedStake2 := Stake{
		ID:          2,
		ArgumentID:  1,
		Type:        StakeUpvote,
		Amount:      sdk.NewInt64Coin(app.StakeDenom, 10),
		Creator:     addr2,
		CreatedTime: ctx.BlockHeader().Time,
		EndTime:     ctx.BlockHeader().Time.Add(time.Hour * 24 * 7),
	}
	// fail if argument doesn't exist
	_, err = k.SubmitUpvote(ctx, 9999, addr)
	assert.Error(t, err)
	assert.Equal(t, ErrorCodeUnknownArgument, err.Code())
	// don't let stake twice
	_, err = k.SubmitUpvote(ctx, argument.ID, addr)
	assert.Error(t, err)
	assert.Equal(t, ErrorCodeDuplicateStake, err.Code())
	_, err = k.SubmitUpvote(ctx, argument.ID, addr2)
	assert.Error(t, err)
	assert.Equal(t, ErrorCodeDuplicateStake, err.Code())

	// user <-> stakes
	user1Stakes := k.UserStakes(ctx, addr)
	user2Stakes := k.UserStakes(ctx, addr2)

	assert.Len(t, user1Stakes, 1)
	assert.Len(t, user2Stakes, 1)

	assert.Equal(t, user1Stakes[0], expectedStake)
	assert.Equal(t, user2Stakes[0], expectedStake2)

	expiringStakes := make([]Stake, 0)

	k.IterateActiveStakeQueue(ctx, ctx.BlockHeader().Time, func(stake Stake) bool {
		expiringStakes = append(expiringStakes, stake)
		return false
	})
	// shouldn't have any expiring stake
	assert.Len(t, expiringStakes, 0)

	period := k.GetParams(ctx).Period
	k.IterateActiveStakeQueue(ctx, ctx.BlockHeader().Time.Add(period), func(stake Stake) bool {
		expiringStakes = append(expiringStakes, stake)
		return false
	})

	assert.Len(t, expiringStakes, 2)
	assert.Equal(t, []Stake{expectedStake, expectedStake2}, expiringStakes)
}

func Test_interest(t *testing.T) {
	ctx, k, _, _ := mockDB()
	amount := sdk.NewInt64Coin(app.StakeDenom, 500000000000000)
	now := time.Now()
	p := k.GetParams(ctx)
	after7days := now.Add(p.Period)
	interest := k.interest(ctx, amount, after7days.Sub(now))
	assert.Equal(t, sdk.NewInt(2397260273973), interest.RoundInt())
}

func Test_splitReward(t *testing.T) {
	ctx, k, _, _ := mockDB()
	amount := sdk.NewInt64Coin(app.StakeDenom, 500000000000000)
	now := time.Now()
	p := k.GetParams(ctx)
	after7days := now.Add(p.Period)
	interest := k.interest(ctx, amount, after7days.Sub(now))
	creatorReward, stakerReward := k.splitReward(ctx, interest)
	expectedCreatorReward := sdk.NewDecFromInt(sdk.NewInt(2397260273973)).
		Mul(sdk.NewDecWithPrec(50, 2))

	assert.True(t, amount.Amount.GT(interest.RoundInt()))
	assert.True(t, interest.RoundInt().GT(creatorReward))
	assert.True(t, interest.RoundInt().GT(stakerReward))
	assert.True(t, creatorReward.Equal(stakerReward))
	assert.Equal(t,
		expectedCreatorReward.RoundInt(),
		creatorReward,
	)
	assert.Equal(t,
		interest.Sub(expectedCreatorReward).RoundInt(),
		stakerReward,
	)
}
