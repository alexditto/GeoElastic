package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"geoelastic/internal/service"
)

type credentialsRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type RegisterUserHandler struct {
	Auth *service.Authenticator
}

// ServeHTTP creates a new user from a username and password. Open to
// anyone — there's no invite/approval step for this project.
func (h *RegisterUserHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}

	id, err := h.Auth.RegisterUser(r.Context(), req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrMissingCredentials):
			w.WriteHeader(http.StatusBadRequest)
		case errors.Is(err, service.ErrUsernameTaken):
			w.WriteHeader(http.StatusConflict)
		default:
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"id": id, "username": req.Username})
}

type IssueTokenHandler struct {
	Auth *service.Authenticator
}

// ServeHTTP exchanges a username/password for a new access token. The raw
// token is only ever returned here, once — the server only ever stores its
// hash, so losing this response means the caller has to issue a new token.
func (h *IssueTokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var req credentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid JSON body: " + err.Error()})
		return
	}

	token, err := h.Auth.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			w.WriteHeader(http.StatusUnauthorized)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"access_token": token})
}
