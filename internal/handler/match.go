package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"fmt"
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
		fmt.Println("Error matching request:", err)
		if errors.Is(err, service.ErrEmptyMatchRequest) {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "at least one field must be provided to match on. Available fields: name, display_name, address, phone_number, location"})
			return
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(resp)
}
