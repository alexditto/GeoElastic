package handler

import (
	"encoding/json"
	"net/http"

	"geoelastic/internal/model"
	"geoelastic/internal/store"
)

type GetAllBusinessesHandler struct {
	Store *store.ElasticsearchStore
}

func (h *GetAllBusinessesHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	businesses, err := h.Store.GetAllBusinesses(r.Context())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(businesses)

}

type GetBusinessByIDHandler struct {
	Store *store.ElasticsearchStore
}

func (h *GetBusinessByIDHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	id := r.PathValue("id")
	if id == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing 'id' path parameter"})
		return
	}

	business, err := h.Store.GetBusinessByID(r.Context(), id)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	if business == nil {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "business not found"})
		return
	}

	json.NewEncoder(w).Encode(business)
}

type CreateBusinessHandler struct {
	Store *store.ElasticsearchStore
}

func (h *CreateBusinessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var business model.Business
	if err := json.NewDecoder(r.Body).Decode(&business); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}
	business.ID = "" // the ID is always Elasticsearch-assigned; ignore any client-supplied value

	id, err := h.Store.CreateBusiness(r.Context(), business)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	business.ID = id
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(business)
}
