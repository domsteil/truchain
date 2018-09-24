package db

import (
	"time"

	ts "github.com/TruStory/truchain/x/truchain/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ============================================================================
// Story operations

// NewStory adds a story to the key-value store
func (k TruKeeper) NewStory(
	ctx sdk.Context,
	body string,
	category ts.StoryCategory,
	creator sdk.AccAddress,
	escrow sdk.AccAddress,
	storyType ts.StoryType,
	voteMaxNum int64,
	voteStart time.Time,
	voteEnd time.Time) (int64, sdk.Error) {

	store := ctx.KVStore(k.storyKey)

	story := ts.Story{
		ID:           k.newID(ctx, k.storyKey),
		Body:         body,
		Category:     category,
		CreatedBlock: ctx.BlockHeight(),
		Creator:      creator,
		Escrow:       escrow,
		State:        ts.Created,
		StoryType:    storyType,
		VoteMaxNum:   voteMaxNum,
		VoteStart:    voteStart,
		VoteEnd:      voteEnd,
	}

	key := generateKey(k.storyKey.String(), story.ID)
	val := k.cdc.MustMarshalBinary(story)
	store.Set(key, val)

	// add story to the active story queue (for in-progress stories)
	k.ActiveStoryQueuePush(ctx, story.ID)

	return story.ID, nil
}

// GetStory gets the story with the given id from the key-value store
func (k TruKeeper) GetStory(ctx sdk.Context, storyID int64) (ts.Story, sdk.Error) {
	store := ctx.KVStore(k.storyKey)
	key := generateKey(k.storyKey.String(), storyID)
	val := store.Get(key)
	if val == nil {
		return ts.Story{}, ts.ErrStoryNotFound(storyID)
	}
	story := &ts.Story{}
	k.cdc.MustUnmarshalBinary(val, story)

	return *story, nil
}

// UpdateStory updates an existing story in the store
func (k TruKeeper) UpdateStory(ctx sdk.Context, story ts.Story) {
	newStory := ts.NewStory(
		story.ID,
		story.BackIDs,
		story.CommentIDs,
		story.EvidenceIDs,
		story.Thread,
		story.VoteIDs,
		story.Body,
		story.Category,
		story.CreatedBlock,
		story.Creator,
		story.Escrow,
		story.Round,
		story.State,
		story.StoryType,
		ctx.BlockHeight(),
		story.Users,
		story.VoteMaxNum,
		story.VoteStart,
		story.VoteEnd)

	store := ctx.KVStore(k.storyKey)
	key := generateKey(k.storyKey.String(), story.ID)
	val := k.cdc.MustMarshalBinary(newStory)
	store.Set(key, val)
}

// ============================================================================
// Actions that can be performed on a story

// VoteStory saves a vote to a story
func (k TruKeeper) VoteStory(ctx sdk.Context, storyID int64, creator sdk.AccAddress, choice bool, amount sdk.Coins) (int64, sdk.Error) {
	story, err := k.GetStory(ctx, storyID)
	if err != nil {
		return -1, err
	}

	// temporarily moves funds from voter to an escrow account until
	// the voting period is over and funds are distributed
	_, err = k.ck.SendCoins(ctx, creator, story.Escrow, amount)
	if err != nil {
		return -1, err
	}

	voteID, err := k.NewVote(ctx, story, amount, creator, choice)

	// add vote id to story
	story.VoteIDs = append(story.VoteIDs, voteID)

	// replace old story with new one in story store
	k.UpdateStory(ctx, story)

	// add vote to vote list
	votes := k.GetActiveVotes(ctx, story.ID)
	votes = append(votes, voteID)
	k.SetActiveVotes(ctx, story.ID, votes)

	return voteID, nil
}
