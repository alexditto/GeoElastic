package model

import "time"

// Token is an issued access token record. TokenHash is the SHA-256 hex
// digest of the raw token handed to the caller once at issuance — the raw
// token itself is never stored, so a stolen index dump can't be replayed
// as a valid credential. Like User, this mirrors the Elasticsearch document
// shape rather than an API response shape; handlers must never serialize a
// Token directly.
type Token struct {
	ID        string     `json:"id,omitempty"`
	UserID    string     `json:"user_id"`
	TokenHash string     `json:"token_hash"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	RevokedAt *time.Time `json:"revoked_at,omitempty"`
}
