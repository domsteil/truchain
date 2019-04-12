package challenge

import (
	"encoding/json"

	app "github.com/TruStory/truchain/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// NewHandler creates a new handler for all challenge messages
func NewHandler(k Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) sdk.Result {
		switch msg := msg.(type) {
		case CreateChallengeMsg:
			return handleCreateChallengeMsg(ctx, k, msg)
		case LikeChallengeArgumentMsg:
			return handleLikeArgumentMsg(ctx, k, msg)
		default:
			return app.ErrMsgHandler(msg)
		}
	}
}

// handles a message to create a challenge
func handleCreateChallengeMsg(
	ctx sdk.Context, k Keeper, msg CreateChallengeMsg) sdk.Result {

	if err := msg.ValidateBasic(); err != nil {
		return err.Result()
	}

	id, err := k.Create(
		ctx, msg.StoryID, msg.Amount, 0, msg.Argument, msg.Creator)
	if err != nil {
		return err.Result()
	}
	story, err := k.storyKeeper.Story(ctx, msg.StoryID)
	if err != nil {
		return err.Result()
	}

	return result(ctx, k, msg.StoryID, id, msg.Creator, story.Creator, msg.Amount)
}

func handleLikeArgumentMsg(ctx sdk.Context, k Keeper, msg LikeChallengeArgumentMsg) sdk.Result {
	if err := msg.ValidateBasic(); err != nil {
		return err.Result()
	}

	challengeID, err := k.LikeArgument(ctx, msg.ArgumentID, msg.Creator, msg.Amount)
	if err != nil {
		return err.Result()
	}

	challenge, err := k.Challenge(ctx, challengeID)
	if err != nil {
		err.Result()
	}
	argument, err := k.argumentKeeper.Argument(ctx, msg.ArgumentID)
	if err != nil {
		return err.Result()
	}
	return result(ctx, k, challenge.StoryID(), challengeID, msg.Creator, argument.Creator, msg.Amount)
}

func result(ctx sdk.Context, k Keeper, storyID, challengeID int64, from, to sdk.AccAddress, amount sdk.Coin) sdk.Result {

	resultData := app.StakeNotificationResult{
		MsgResult: app.MsgResult{ID: challengeID},
		Amount:    amount,
		StoryID:   storyID,
		From:      from,
		To:        to,
	}

	resultBytes, jsonErr := json.Marshal(resultData)
	if jsonErr != nil {
		panic(jsonErr)
	}

	return sdk.Result{
		Data: resultBytes,
		Tags: app.PushTag,
	}
}
