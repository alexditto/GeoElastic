package store

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"geoelastic/internal/model"
)

// ErrDuplicateUsername is returned when CreateUser is asked to create a
// user whose username is already taken.
var ErrDuplicateUsername = errors.New("username already exists")

// CreateUser indexes a new user, keyed by username, and returns the
// document ID Elasticsearch assigned to it. u.PasswordHash must already be
// a bcrypt hash — the store layer never hashes or verifies passwords
// itself.
//
// Using the username as the document ID and op_type "create" makes
// uniqueness an atomic property of the write itself (ES rejects a second
// create for the same ID with 409), rather than a separate look-up-then-
// write check racing against Elasticsearch's near-real-time search — a
// just-indexed document isn't guaranteed searchable until the next index
// refresh (every ~1s by default), so two rapid registrations could both
// pass a "does this username exist" search before either was visible.
func (s *ElasticsearchStore) CreateUser(ctx context.Context, u model.User) (string, error) {
	body, err := json.Marshal(u)
	if err != nil {
		return "", fmt.Errorf("encoding user: %w", err)
	}

	res, err := s.client.Index(
		userAlias,
		bytes.NewReader(body),
		s.client.Index.WithDocumentID(u.Username),
		s.client.Index.WithOpType("create"),
		s.client.Index.WithContext(ctx),
	)
	if err != nil {
		return "", fmt.Errorf("indexing user: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusConflict {
		return "", ErrDuplicateUsername
	}
	if res.IsError() {
		return "", fmt.Errorf("indexing user: %s", res.Status())
	}

	var indexed struct {
		ID string `json:"_id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&indexed); err != nil {
		return "", fmt.Errorf("decoding index response: %w", err)
	}

	return indexed.ID, nil
}

// GetUserByUsername looks up a user by their exact username, returning nil
// if no user has that username. Since CreateUser keys the document by
// username, this is a direct Get rather than a _search query — Get is
// real-time (Elasticsearch consults the in-flight translog, not just the
// last-refreshed index), so a user can log in immediately after
// registering instead of possibly missing an index refresh cycle.
func (s *ElasticsearchStore) GetUserByUsername(ctx context.Context, username string) (*model.User, error) {
	res, err := s.client.Get(
		userAlias,
		username,
		s.client.Get.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("getting user by username: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		if res.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("getting user by username: %s", res.Status())
	}

	var result struct {
		ID     string     `json:"_id"`
		Source model.User `json:"_source"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding get response: %w", err)
	}

	result.Source.ID = result.ID
	return &result.Source, nil
}
