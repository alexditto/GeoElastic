package main

import (
	"log"
	"net/http"

	"geoelastic/internal/config"
	"geoelastic/internal/handler"
	"geoelastic/internal/middleware"
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
	creator := &service.BusinessCreator{Store: es}
	auth := &service.Authenticator{Store: es}

	mux := http.NewServeMux()
	mux.Handle("GET /health", &handler.HealthHandler{Store: es})
	mux.Handle("POST /users", &handler.RegisterUserHandler{Auth: auth})
	mux.Handle("POST /tokens", &handler.IssueTokenHandler{Auth: auth})

	mux.Handle("GET /businesses", middleware.RequireAuth(auth, &handler.GetAllBusinessesHandler{Store: es}))
	mux.Handle("POST /businesses", middleware.RequireAuth(auth, &handler.CreateBusinessHandler{Creator: creator}))
	mux.Handle("GET /business/{id}", middleware.RequireAuth(auth, &handler.GetBusinessByIDHandler{Store: es}))
	mux.Handle("POST /businesses/match", middleware.RequireAuth(auth, &handler.MatchHandler{Matcher: matcher}))

	addr := ":" + cfg.ServerPort
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}
