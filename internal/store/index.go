package store

import (
	"bytes"
	"context"
	_ "embed"
	"fmt"
	"net/http"
)

//go:embed mappings/businesses_v1.json
var businessMappingV1 []byte

//go:embed mappings/users_v1.json
var userMappingV1 []byte

//go:embed mappings/tokens_v1.json
var tokenMappingV1 []byte

const (
	businessIndexV1 = "businesses_v1"
	businessAlias   = "businesses"

	userIndexV1 = "users_v1"
	userAlias   = "users"

	tokenIndexV1 = "tokens_v1"
	tokenAlias   = "tokens"
)

// EnsureBusinessIndex creates the versioned businesses index and its alias
// if they don't already exist yet. Application code should always query the
// businessAlias, not the versioned index name directly — a future mapping
// change means creating businesses_v2, reindexing into it, and swapping the
// alias, rather than altering this index in place (most mapping changes
// can't be applied to a live index).
func (s *ElasticsearchStore) EnsureBusinessIndex(ctx context.Context) error {
	return s.ensureIndex(ctx, businessIndexV1, businessAlias, businessMappingV1)
}

// EnsureUserIndex creates the versioned users index and its alias if they
// don't already exist yet. See EnsureBusinessIndex for the versioning
// convention this follows.
func (s *ElasticsearchStore) EnsureUserIndex(ctx context.Context) error {
	return s.ensureIndex(ctx, userIndexV1, userAlias, userMappingV1)
}

// EnsureTokenIndex creates the versioned tokens index and its alias if they
// don't already exist yet. See EnsureBusinessIndex for the versioning
// convention this follows.
func (s *ElasticsearchStore) EnsureTokenIndex(ctx context.Context) error {
	return s.ensureIndex(ctx, tokenIndexV1, tokenAlias, tokenMappingV1)
}

// ensureIndex creates the given versioned index from mapping, and points
// alias at it, if they don't already exist.
func (s *ElasticsearchStore) ensureIndex(ctx context.Context, index, alias string, mapping []byte) error {
	indexRes, err := s.client.Indices.Exists(
		[]string{index},
		s.client.Indices.Exists.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("checking for index %q: %w", index, err)
	}
	defer indexRes.Body.Close()

	if indexRes.StatusCode == http.StatusNotFound {
		createRes, err := s.client.Indices.Create(
			index,
			s.client.Indices.Create.WithBody(bytes.NewReader(mapping)),
			s.client.Indices.Create.WithContext(ctx),
		)
		if err != nil {
			return fmt.Errorf("creating index %q: %w", index, err)
		}
		defer createRes.Body.Close()
		if createRes.IsError() {
			return fmt.Errorf("creating index %q: %s", index, createRes.Status())
		}
	}

	aliasRes, err := s.client.Indices.ExistsAlias(
		[]string{alias},
		s.client.Indices.ExistsAlias.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("checking for alias %q: %w", alias, err)
	}
	defer aliasRes.Body.Close()

	if aliasRes.StatusCode == http.StatusNotFound {
		putRes, err := s.client.Indices.PutAlias(
			[]string{index},
			alias,
			s.client.Indices.PutAlias.WithContext(ctx),
		)
		if err != nil {
			return fmt.Errorf("creating alias %q: %w", alias, err)
		}
		defer putRes.Body.Close()
		if putRes.IsError() {
			return fmt.Errorf("creating alias %q: %s", alias, putRes.Status())
		}
	}

	return nil
}
