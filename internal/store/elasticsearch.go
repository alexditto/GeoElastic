// Package store holds code that talks directly to Elasticsearch — the
// equivalent layer to what Eloquent's query builder does for you in
// Laravel, hand-written here since there's no ES-flavored ORM for Go.
package store

import (
	"context"
	"fmt"

	"github.com/elastic/go-elasticsearch/v9"
)

type ElasticsearchStore struct {
	client *elasticsearch.Client
}

func NewElasticsearchStore(url, apiKey string) (*ElasticsearchStore, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{url},
		APIKey:    apiKey,
	}

	client, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating elasticsearch client: %w", err)
	}

	return &ElasticsearchStore{client: client}, nil
}

// Ping confirms Elasticsearch is reachable and the credentials are valid.
func (s *ElasticsearchStore) Ping(ctx context.Context) error {
	res, err := s.client.Info(s.client.Info.WithContext(ctx))
	if err != nil {
		return fmt.Errorf("pinging elasticsearch: %w", err)
	}
	defer res.Body.Close()

	if res.IsError() {
		return fmt.Errorf("elasticsearch returned an error status: %s", res.Status())
	}

	return nil
}
