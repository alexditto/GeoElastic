package main

import (
	"log"
	"net/http"

	"geoelastic/internal/config"
	"geoelastic/internal/handler"
	"geoelastic/internal/service"
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

	matcher := &service.BusinessMatcher{Store: es}

	mux := http.NewServeMux()
	mux.Handle("GET /health", &handler.HealthHandler{Store: es})
	mux.Handle("GET /businesses", &handler.GetAllBusinessesHandler{Store: es})
	mux.Handle("POST /businesses", &handler.CreateBusinessHandler{Store: es})
	mux.Handle("GET /business/{id}", &handler.GetBusinessByIDHandler{Store: es})
	mux.Handle("POST /businesses/match", &handler.MatchHandler{Matcher: matcher})

	addr := ":" + cfg.ServerPort
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
