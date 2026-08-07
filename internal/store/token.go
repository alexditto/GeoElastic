package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"geoelastic/internal/model"
)

// CreateToken indexes a new access token record and returns the document ID
// Elasticsearch assigned to it. t.TokenHash must already be hashed — the
// store layer never generates or hashes tokens itself.
func (s *ElasticsearchStore) CreateToken(ctx context.Context, t model.Token) (string, error) {
	body, err := json.Marshal(t)
	if err != nil {
		return "", fmt.Errorf("encoding token: %w", err)
	}

	res, err := s.client.Index(
		tokenAlias,
		bytes.NewReader(body),
		s.client.Index.WithContext(ctx),
	)
	if err != nil {
		return "", fmt.Errorf("indexing token: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return "", fmt.Errorf("indexing token: %s", res.Status())
	}

	var indexed struct {
		ID string `json:"_id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&indexed); err != nil {
		return "", fmt.Errorf("decoding index response: %w", err)
	}

	return indexed.ID, nil
}

// GetTokenByHash looks up a token by the SHA-256 hash of its raw value,
// returning nil if no token has that hash.
func (s *ElasticsearchStore) GetTokenByHash(ctx context.Context, tokenHash string) (*model.Token, error) {
	body := map[string]interface{}{
		"query": map[string]interface{}{
			"term": map[string]interface{}{"token_hash": tokenHash},
		},
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("encoding search query: %w", err)
	}

	res, err := s.client.Search(
		s.client.Search.WithContext(ctx),
		s.client.Search.WithIndex(tokenAlias),
		s.client.Search.WithBody(bytes.NewReader(encoded)),
		s.client.Search.WithSize(1),
	)
	if err != nil {
		return nil, fmt.Errorf("searching tokens: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("searching tokens: %s", res.Status())
	}

	var result struct {
		Hits struct {
			Hits []struct {
				ID     string      `json:"_id"`
				Source model.Token `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding search response: %w", err)
	}

	if len(result.Hits.Hits) == 0 {
		return nil, nil
	}

	token := result.Hits.Hits[0].Source
	token.ID = result.Hits.Hits[0].ID
	return &token, nil
}

// RevokeToken sets revoked_at on the token with the given document ID,
// leaving its record in place as an audit trail rather than deleting it.
func (s *ElasticsearchStore) RevokeToken(ctx context.Context, id string, revokedAt time.Time) error {
	body := map[string]interface{}{
		"doc": map[string]interface{}{
			"revoked_at": revokedAt,
		},
	}

	encoded, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("encoding token update: %w", err)
	}

	res, err := s.client.Update(
		tokenAlias,
		id,
		bytes.NewReader(encoded),
		s.client.Update.WithContext(ctx),
	)
	if err != nil {
		return fmt.Errorf("revoking token: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("revoking token: %s", res.Status())
	}

	return nil
}
