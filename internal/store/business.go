package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"geoelastic/internal/model"
)

func (s *ElasticsearchStore) GetAllBusinesses(ctx context.Context) ([]model.Business, error) {
	res, err := s.client.Search(
		s.client.Search.WithContext(ctx),
		s.client.Search.WithIndex(businessAlias),
		s.client.Search.WithSize(1000), // Adjust the size as needed
	)
	if err != nil {
		return nil, fmt.Errorf("searching businesses: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return nil, fmt.Errorf("searching businesses: %s", res.Status())
	}

	var result struct {
		Hits struct {
			Hits []struct {
				ID     string         `json:"_id"`
				Source model.Business `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding search response: %w", err)
	}

	businesses := make([]model.Business, len(result.Hits.Hits))
	for i, hit := range result.Hits.Hits {
		hit.Source.ID = hit.ID
		businesses[i] = hit.Source
	}
	return businesses, nil
}

func (s *ElasticsearchStore) GetBusinessByID(ctx context.Context, id string) (*model.Business, error) {
	res, err := s.client.Get(
		businessAlias,
		id,
		s.client.Get.WithContext(ctx),
	)
	if err != nil {
		return nil, fmt.Errorf("getting business by ID: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		if res.StatusCode == 404 {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("getting business by ID: %s", res.Status())
	}

	var result struct {
		ID     string         `json:"_id"`
		Source model.Business `json:"_source"`
	}
	if err := json.NewDecoder(res.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding get response: %w", err)
	}

	result.Source.ID = result.ID
	return &result.Source, nil
}

// CreateBusiness indexes a new business document into the businesses alias
// and returns the document ID Elasticsearch assigned to it.
func (s *ElasticsearchStore) CreateBusiness(ctx context.Context, b model.Business) (string, error) {
	body, err := json.Marshal(b)
	if err != nil {
		return "", fmt.Errorf("encoding business: %w", err)
	}

	res, err := s.client.Index(
		businessAlias,
		bytes.NewReader(body),
		s.client.Index.WithContext(ctx),
	)
	if err != nil {
		return "", fmt.Errorf("indexing business: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return "", fmt.Errorf("indexing business: %s", res.Status())
	}

	var indexed struct {
		ID string `json:"_id"`
	}
	if err := json.NewDecoder(res.Body).Decode(&indexed); err != nil {
		return "", fmt.Errorf("decoding index response: %w", err)
	}

	return indexed.ID, nil
}
