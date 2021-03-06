package kettei

import (
	"context"
	"errors"
)

type (
	Strategy int

	DecisionMakerConfig struct {
		Voters                             []Voter
		Strategy                           Strategy
		AllowIfAllAbstainDecisions         bool
		AllowIfEqualGrantedDeniedDecisions bool
	}

	DecisionMaker struct {
		voters                             []Voter
		strategy                           Strategy
		allowIfAllAbstainDecisions         bool
		allowIfEqualGrantedDeniedDecisions bool
	}
)

const (
	StrategyAffirmative = iota
	StrategyConsensus
	StrategyUnanimous
)

var (
	ErrInvalidStrategy = errors.New("invalid strategy")
)

func NewDecisionMaker(config DecisionMakerConfig) *DecisionMaker {
	return &DecisionMaker{
		voters:                             config.Voters,
		strategy:                           config.Strategy,
		allowIfAllAbstainDecisions:         config.AllowIfAllAbstainDecisions,
		allowIfEqualGrantedDeniedDecisions: config.AllowIfEqualGrantedDeniedDecisions,
	}
}

func NewDefaultDecisionMaker(voters ...Voter) *DecisionMaker {
	return NewDecisionMaker(DecisionMakerConfig{
		Voters:                     voters,
		Strategy:                   StrategyUnanimous,
		AllowIfAllAbstainDecisions: true,
	})
}

// Decides whether the access is possible or not.
func (maker *DecisionMaker) Decide(ctx context.Context, attributes []string, subject interface{}) (bool, error) {
	switch maker.strategy {
	case StrategyAffirmative:
		return maker.decideAffirmative(ctx, attributes, subject)
	case StrategyConsensus:
		return maker.decideConsensus(ctx, attributes, subject)
	case StrategyUnanimous:
		return maker.decideUnanimous(ctx, attributes, subject)
	default:
		return false, ErrInvalidStrategy
	}
}

// Grants access if any voter returns an affirmative response.
//
// If all voters abstained from voting, the decision will be based on the allowIfAllAbstainDecisions property value
// (defaults to false).
func (maker *DecisionMaker) decideAffirmative(ctx context.Context, attributes []string, subject interface{}) (bool, error) {
	var deny int

	for _, voter := range maker.voters {
		result, err := vote(voter, ctx, attributes, subject)
		if err != nil {
			return false, err
		}

		switch result {
		case AccessGranted:
			return true, nil
		case AccessDenied:
			deny += 1
			break
		default:
			break
		}
	}

	if deny > 0 {
		return false, nil
	}

	return maker.allowIfAllAbstainDecisions, nil
}

// Grants access if there is consensus of granted against denied responses.
//
// Consensus means majority-rule (ignoring abstains) rather than unanimous agreement (ignoring abstains).
// If you require unanimity, see UnanimousBased.
//
// If there were an equal number of grant and deny votes, the decision will be based on the
// allowIfEqualGrantedDeniedDecisions property value (defaults to true).
//
// If all voters abstained from voting, the decision will be based on the allowIfAllAbstainDecisions property value
// (defaults to false).
func (maker *DecisionMaker) decideConsensus(ctx context.Context, attributes []string, subject interface{}) (bool, error) {
	var grant int
	var deny int

	for _, voter := range maker.voters {
		result, err := vote(voter, ctx, attributes, subject)
		if err != nil {
			return false, err
		}

		switch result {
		case AccessGranted:
			grant += 1
			break
		case AccessDenied:
			deny += 1
			break
		default:
			break
		}
	}

	if grant > deny {
		return true, nil
	}

	if deny > grant {
		return false, nil
	}

	if grant > 0 {
		return maker.allowIfEqualGrantedDeniedDecisions, nil
	}

	return maker.allowIfAllAbstainDecisions, nil
}

// Grants access if only grant (or abstain) votes were received.
//
// If all voters abstained from voting, the decision will be based on the allowIfAllAbstainDecisions property value
// (defaults to false).
func (maker *DecisionMaker) decideUnanimous(ctx context.Context, attributes []string, subject interface{}) (bool, error) {
	var grant int

	for _, voter := range maker.voters {
		for _, attribute := range attributes {
			result, err := vote(voter, ctx, []string{attribute}, subject)
			if err != nil {
				return false, err
			}

			switch result {
			case AccessGranted:
				grant += 1
				break
			case AccessDenied:
				return false, nil
			default:
				break
			}
		}
	}

	if grant > 0 {
		return true, nil
	}

	return maker.allowIfAllAbstainDecisions, nil
}
