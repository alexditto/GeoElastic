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

	if err := es.EnsureBusinessIndex(context.Background()); err != nil {
		log.Fatalf("ensuring business index: %v", err)
	}

	log.Println("businesses index and alias are up to date")
}
