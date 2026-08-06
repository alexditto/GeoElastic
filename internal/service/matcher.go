// Package service holds decision logic that sits between HTTP handlers and
// the store — the layer Eloquent-style "business rules" or an Action class
// would occupy in Laravel, but with no framework machinery behind it.
package service

import (
	"context"
	"errors"
	"fmt"

	"geoelastic/internal/model"
	"geoelastic/internal/store"
)

// ErrEmptyMatchRequest is returned when a MatchRequest has no fields set at
// all — there's nothing for the matcher to build a query from.
var ErrEmptyMatchRequest = errors.New("at least one field must be provided to match on")

// ScoredBusiness is a Business plus how closely it matched a MatchRequest.
// Embedding Business means its fields serialize flattened, alongside score,
// rather than nested under a "business" key.
type ScoredBusiness struct {
	model.Business
	Score float64 `json:"score"`
}

type MatchResponse struct {
	Match      string           `json:"match"`
	Businesses []ScoredBusiness `json:"business"`
}

type BusinessMatcher struct {
	Store *store.ElasticsearchStore
}

// Match looks for an exact match first; if that doesn't resolve to exactly
// one business, it falls back to a ranked fuzzy search. Fuzzy scores are
// normalized against the top hit in the result set (best match = 1.0) —
// Elasticsearch's raw relevance scores aren't comparable across searches,
// so this makes them at least comparable within one response.
func (m *BusinessMatcher) Match(ctx context.Context, req model.MatchRequest) (MatchResponse, error) {
	if req.IsEmpty() {
		return MatchResponse{}, ErrEmptyMatchRequest
	}

	exact, err := m.Store.SearchExactBusiness(ctx, req)
	if err != nil {
		return MatchResponse{}, fmt.Errorf("searching for exact match: %w", err)
	}
	if len(exact) == 1 {
		return MatchResponse{
			Match:      "exact",
			Businesses: []ScoredBusiness{{Business: exact[0], Score: 1.0}},
		}, nil
	}

	fuzzy, err := m.Store.SearchFuzzyBusiness(ctx, req)
	if err != nil {
		return MatchResponse{}, fmt.Errorf("searching for fuzzy match: %w", err)
	}
	if len(fuzzy) == 0 {
		return MatchResponse{Match: "none", Businesses: []ScoredBusiness{}}, nil
	}

	topScore := fuzzy[0].Score // Elasticsearch already sorts hits by score, descending.
	businesses := make([]ScoredBusiness, len(fuzzy))
	for i, hit := range fuzzy {
		businesses[i] = ScoredBusiness{Business: hit.Business, Score: hit.Score / topScore}
	}

	return MatchResponse{Match: "fuzzy", Businesses: businesses}, nil
}
