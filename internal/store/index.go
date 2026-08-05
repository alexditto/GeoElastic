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

const (
	businessIndexV1 = "businesses_v1"
	businessAlias   = "businesses"
)

// EnsureBusinessIndex creates the versioned businesses index and its alias
// if they don't already exist yet. Application code should always query the
// businessAlias, not the versioned index name directly — a future mapping
// change means creating businesses_v2, reindexing into it, and swapping the
// alias, rather than altering this index in place (most mapping changes
// can't be applied to a live index).
func (s *ElasticsearchStore) EnsureBusinessIndex(ctx context.Context) error {
	indexRes, err := s.client.Indices.Exists(
		[]string{businessIndexV1},
		s.client.Indices.Exists.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("checking for index %q: %w", businessIndexV1, err)
	}
	defer indexRes.Body.Close()

	if indexRes.StatusCode == http.StatusNotFound {
		createRes, err := s.client.Indices.Create(
			businessIndexV1,
			s.client.Indices.Create.WithBody(bytes.NewReader(businessMappingV1)),
			s.client.Indices.Create.WithContext(ctx),
		)
		if err != nil {
			return fmt.Errorf("creating index %q: %w", businessIndexV1, err)
		}
		defer createRes.Body.Close()
		if createRes.IsError() {
			return fmt.Errorf("creating index %q: %s", businessIndexV1, createRes.Status())
		}
	}

	aliasRes, err := s.client.Indices.ExistsAlias(
		[]string{businessAlias},
		s.client.Indices.ExistsAlias.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("checking for alias %q: %w", businessAlias, err)
	}
	defer aliasRes.Body.Close()

	if aliasRes.StatusCode == http.StatusNotFound {
		putRes, err := s.client.Indices.PutAlias(
			[]string{businessIndexV1},
			businessAlias,
			s.client.Indices.PutAlias.WithContext(ctx),
		)
		if err != nil {
			return fmt.Errorf("creating alias %q: %w", businessAlias, err)
		}
		defer putRes.Body.Close()
		if putRes.IsError() {
			return fmt.Errorf("creating alias %q: %s", businessAlias, putRes.Status())
		}
	}

	return nil
}
