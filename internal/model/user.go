package model

import "time"

// User is an account that can authenticate against the API and be issued
// access tokens. PasswordHash is a bcrypt hash, never the plaintext
// password. This struct mirrors the Elasticsearch document shape rather
// than an API response shape — handlers must never serialize a User
// directly, since that would leak PasswordHash to the client.
type User struct {
	ID           string    `json:"id,omitempty"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"password_hash"`
	CreatedAt    time.Time `json:"created_at"`
}
