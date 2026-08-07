package main

import (
	"context"
	"log"

	"geoelastic/internal/config"
	"geoelastic/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("loading config: %v", err)
	}

	es, err := store.NewElasticsearchStore(cfg.ElasticsearchURL, cfg.ElasticsearchAPIKey)
	if err != nil {
		log.Fatalf("connecting to elasticsearch: %v", err)
	}

	ctx := context.Background()

	if err := es.EnsureBusinessIndex(ctx); err != nil {
		log.Fatalf("ensuring business index: %v", err)
	}
	if err := es.EnsureUserIndex(ctx); err != nil {
		log.Fatalf("ensuring user index: %v", err)
	}
	if err := es.EnsureTokenIndex(ctx); err != nil {
		log.Fatalf("ensuring token index: %v", err)
	}

	log.Println("businesses, users, and tokens indices and aliases are up to date")
}
