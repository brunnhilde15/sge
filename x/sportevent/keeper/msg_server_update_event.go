package keeper

import (
	"context"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/sge-network/sge/x/sportevent/types"
)

// UpdateEvent accepts ticket containing multiple update events and return batch response after processing
func (k msgServer) UpdateEvent(goCtx context.Context, msg *types.MsgUpdateEvent) (*types.SportResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	var updateEvent types.SportEvent
	if err := k.dvmKeeper.VerifyTicketUnmarshal(goCtx, msg.Ticket, &updateEvent); err != nil {
		return nil, sdkerrors.Wrapf(types.ErrInVerification, "%s", err)
	}

	sportEvent, found := k.Keeper.GetSportEvent(ctx, updateEvent.GetUID())
	if !found {
		return nil, types.ErrEventNotFound
	}

	// if update event is not valid so it is failed
	if err := k.validateEventUpdate(ctx, updateEvent, sportEvent); err != nil {
		return nil, sdkerrors.Wrap(err, "validate update data")
	}

	// if the status is still pending so it is failed
	if sportEvent.Status != types.SportEventStatus_STATUS_PENDING {
		return nil, types.ErrCanNotBeAltered
	}

	sportEvent = types.SportEvent{
		UID:            sportEvent.UID,
		OddsUIDs:       sportEvent.OddsUIDs,
		WinnerOddsUIDs: sportEvent.WinnerOddsUIDs,
		Status:         sportEvent.Status,
		ResolutionTS:   sportEvent.ResolutionTS,
		Creator:        sportEvent.Creator,
		StartTS:        updateEvent.StartTS,
		EndTS:          updateEvent.EndTS,
		BetConstraints: &types.EventBetConstraints{
			MinAmount: updateEvent.BetConstraints.MinAmount,
			BetFee:    updateEvent.BetConstraints.BetFee,
		},
		Active: updateEvent.Active,
	}
	// the update event is successful so update the module state
	k.Keeper.SetSportEvent(ctx, sportEvent)

	response := &types.SportResponse{
		Data: &sportEvent,
	}
	emitTransactionEvent(ctx, types.TypeMsgUpdateSportEvents, response, msg.Creator)
	return response, nil
}

// validateEventUpdate validates individual event acceptability
func (k msgServer) validateEventUpdate(ctx sdk.Context, event, previousEvent types.SportEvent) error {
	if event.EndTS <= uint64(ctx.BlockTime().Unix()) {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid end timestamp for the sport event")
	}

	if event.StartTS >= event.EndTS || event.StartTS == 0 {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "invalid start timestamp for the sport event")
	}

	params := k.GetParams(ctx)

	if event.BetConstraints == nil {
		event.BetConstraints = previousEvent.BetConstraints
		return nil
	}

	//init individual params if any one of them is nil
	initEventConstrains(event, previousEvent)

	//check the validity constraints
	if !(event.BetConstraints.BetFee.IsLT(params.EventMinBetFee) || event.BetConstraints.BetFee.IsEqual(params.EventMinBetFee)) {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "event bet fee is out of threshold limit")
	}

	if event.BetConstraints.MinAmount.IsNegative() {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "event min amount can not be negative")
	}
	if event.BetConstraints.MinAmount.LT(params.EventMinBetAmount) {
		return sdkerrors.Wrapf(sdkerrors.ErrInvalidRequest, "event min bet amount is less than threshold")
	}

	return nil
}

func initEventConstrains(event, previousEvent types.SportEvent) {
	//init individual params if any one of them is nil
	if event.BetConstraints.BetFee.IsNil() {
		event.BetConstraints.BetFee = previousEvent.BetConstraints.BetFee
	}
	if event.BetConstraints.MinAmount.IsNil() {
		event.BetConstraints.MinAmount = previousEvent.BetConstraints.MinAmount
	}
}
