package keeper

import (
	"context"
	"fmt"

	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	"github.com/cosmos/cosmos-sdk/x/gov/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = Keeper{}

// Proposal returns proposal details based on ProposalID
func (q Keeper) Proposal(c context.Context, req *types.QueryProposalRequest) (*types.QueryProposalResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)

	if req.ProposalId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request")
	}

	proposal, found := q.GetProposal(ctx, req.ProposalId)
	if !found {
		return &types.QueryProposalResponse{}, sdkerrors.Wrapf(types.ErrUnknownProposal, "%d", req.ProposalId)
	}

	return &types.QueryProposalResponse{Proposal: proposal}, nil
}

// Proposals implements the Query/Proposals gRPC method
func (q Keeper) Proposals(c context.Context, req *types.QueryProposalsRequest) (*types.QueryProposalsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	var proposals types.Proposals
	ctx := sdk.UnwrapSDKContext(c)

	// TODO: update to use conditional store
	store := ctx.KVStore(q.storeKey)
	proposalStore := prefix.NewStore(store, types.ProposalsKeyPrefix)

	res, err := query.Paginate(proposalStore, req.Req, func(key []byte, value []byte) error {
		var result types.Proposal
		err := q.cdc.UnmarshalBinaryBare(value, &result)
		if err != nil {
			return err
		}
		proposals = append(proposals, result)
		return nil
	})

	if err != nil {
		return &types.QueryProposalsResponse{}, err
	}

	return &types.QueryProposalsResponse{Proposals: proposals, Res: res}, nil
}

// Vote returns Voted information based on proposalID, voterAddr
func (q Keeper) Vote(c context.Context, req *types.QueryVoteRequest) (*types.QueryVoteResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	ctx := sdk.UnwrapSDKContext(c)

	if req.ProposalId == 0 || req.Voter == nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request")
	}

	vote, found := q.GetVote(ctx, req.ProposalId, req.Voter)
	if !found {
		return &types.QueryVoteResponse{}, status.Errorf(codes.InvalidArgument,
			fmt.Sprintf("Voter: %v not found for proposal: %v", req.Voter, req.ProposalId))
	}

	return &types.QueryVoteResponse{Vote: vote}, nil
}

// Votes returns single proposal's votes
func (q Keeper) Votes(c context.Context, req *types.QueryProposalRequest) (*types.QueryVotesResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	if req.ProposalId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request")
	}

	var votes types.Votes
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(q.storeKey)
	votesStore := prefix.NewStore(store, types.VotesKey(req.ProposalId))

	res, err := query.Paginate(votesStore, req.Req, func(key []byte, value []byte) error {
		var result types.Vote
		err := q.cdc.UnmarshalBinaryBare(value, &result)
		if err != nil {
			return err
		}
		votes = append(votes, result)
		return nil
	})

	if err != nil {
		return &types.QueryVotesResponse{}, err
	}

	return &types.QueryVotesResponse{Votes: votes, Res: res}, nil
}

// Params queries all params
func (q Keeper) Params(c context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	if req.ParamsType == "" {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)

	switch req.ParamsType {
	case types.ParamDeposit:
		depositParmas := q.GetDepositParams(ctx)

		return &types.QueryParamsResponse{DepositParams: depositParmas}, nil

	case types.ParamVoting:
		votingParmas := q.GetVotingParams(ctx)

		return &types.QueryParamsResponse{VotingParams: votingParmas}, nil

	case types.ParamTallying:
		tallyParams := q.GetTallyParams(ctx)

		return &types.QueryParamsResponse{TallyParams: tallyParams}, nil

	default:
		return nil, sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "%s is not a valid query request path", req.ParamsType)
	}
}

// Deposit queries single deposit information based proposalID, depositAddr
func (q Keeper) Deposit(c context.Context, req *types.QueryDepositRequest) (*types.QueryDepositResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	if req.ProposalId == 0 || req.Depositor == nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)

	deposit, found := q.GetDeposit(ctx, req.ProposalId, req.Depositor)
	if !found {
		return &types.QueryDepositResponse{}, status.Errorf(codes.InvalidArgument,
			fmt.Sprintf("Depositer: %v not found for proposal: %v", req.Depositor, req.ProposalId))
	}

	return &types.QueryDepositResponse{Deposit: deposit}, nil
}

// Deposits returns single proposal's all deposits
func (q Keeper) Deposits(c context.Context, req *types.QueryProposalRequest) (*types.QueryDepositsResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	if req.ProposalId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request")
	}

	var deposits types.Deposits
	ctx := sdk.UnwrapSDKContext(c)

	store := ctx.KVStore(q.storeKey)
	depositStore := prefix.NewStore(store, types.DepositsKey(req.ProposalId))

	res, err := query.Paginate(depositStore, req.Req, func(key []byte, value []byte) error {
		var result types.Deposit
		err := q.cdc.UnmarshalBinaryBare(value, &result)
		if err != nil {
			return err
		}
		deposits = append(deposits, result)
		return nil
	})

	if err != nil {
		return &types.QueryDepositsResponse{}, err
	}

	return &types.QueryDepositsResponse{Deposits: deposits, Res: res}, nil
}

// TallyResult queries the tally of a proposal vote
func (q Keeper) TallyResult(c context.Context, req *types.QueryProposalRequest) (*types.QueryTallyResponse, error) {
	if req == nil {
		return nil, status.Errorf(codes.InvalidArgument, "empty request")
	}

	if req.ProposalId == 0 {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(c)

	proposal, ok := q.GetProposal(ctx, req.ProposalId)
	if !ok {
		return nil, sdkerrors.Wrapf(types.ErrUnknownProposal, "%d", req.ProposalId)
	}

	var tallyResult types.TallyResult

	switch {
	case proposal.Status == types.StatusDepositPeriod:
		tallyResult = types.EmptyTallyResult()

	case proposal.Status == types.StatusPassed || proposal.Status == types.StatusRejected:
		tallyResult = proposal.FinalTallyResult

	default:
		// proposal is in voting period
		_, _, tallyResult = q.Tally(ctx, proposal)
	}

	return &types.QueryTallyResponse{Tally: tallyResult}, nil
}