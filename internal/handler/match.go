package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"geoelastic/internal/model"
	"geoelastic/internal/service"
)

type MatchHandler struct {
	Matcher *service.BusinessMatcher
}

func (h *MatchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req model.MatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}

	resp, err := h.Matcher.Match(r.Context(), req)
	if err != nil {
		if errors.Is(err, service.ErrEmptyMatchRequest) {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(resp)
}
