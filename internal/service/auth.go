package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"

	"geoelastic/internal/model"
	"geoelastic/internal/store"
)

// TokenTTL is how long a newly issued access token stays valid.
const TokenTTL = 30 * 24 * time.Hour

// ErrInvalidCredentials is returned when a username/password pair doesn't
// match an existing user.
var ErrInvalidCredentials = errors.New("invalid username or password")

// ErrMissingCredentials is returned when registration is attempted with a
// blank username or password.
var ErrMissingCredentials = errors.New("username and password are required")

// ErrUsernameTaken is returned when registration is attempted with a
// username that already belongs to another user — Elasticsearch has no
// native uniqueness constraint, so this has to be checked explicitly.
var ErrUsernameTaken = errors.New("username is already taken")

// ErrInvalidToken is returned for any access token that isn't currently
// usable. Unknown, expired, and revoked tokens are deliberately collapsed
// into this one error — which of the three is true isn't something an API
// caller needs to know, and not distinguishing avoids handing a token
// prober information about which tokens exist.
var ErrInvalidToken = errors.New("invalid or expired token")

// Authenticator registers users and issues, validates, and revokes their
// access tokens.
type Authenticator struct {
	Store *store.ElasticsearchStore
}

// RegisterUser creates a new user, hashing password with bcrypt before it
// ever reaches the store. Uniqueness is enforced atomically by the store
// (see store.CreateUser) rather than by checking first, which would race.
func (a *Authenticator) RegisterUser(ctx context.Context, username, password string) (string, error) {
	if username == "" || password == "" {
		return "", ErrMissingCredentials
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}

	id, err := a.Store.CreateUser(ctx, model.User{
		Username:     username,
		PasswordHash: string(hash),
		CreatedAt:    time.Now(),
	})
	if err != nil {
		if errors.Is(err, store.ErrDuplicateUsername) {
			return "", ErrUsernameTaken
		}
		return "", fmt.Errorf("creating user: %w", err)
	}
	return id, nil
}

// Login verifies a username/password against the stored user and, if it
// matches, issues a new access token for that user.
func (a *Authenticator) Login(ctx context.Context, username, password string) (string, error) {
	user, err := a.Store.GetUserByUsername(ctx, username)
	if err != nil {
		return "", fmt.Errorf("looking up user: %w", err)
	}
	if user == nil {
		return "", ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}

	return a.issueToken(ctx, user.ID)
}

// Authenticate resolves a raw bearer token to the ID of the user who owns
// it, rejecting tokens that don't exist, have expired, or have been
// revoked.
func (a *Authenticator) Authenticate(ctx context.Context, rawToken string) (string, error) {
	token, err := a.Store.GetTokenByHash(ctx, hashToken(rawToken))
	if err != nil {
		return "", fmt.Errorf("looking up token: %w", err)
	}
	if token == nil || token.RevokedAt != nil || time.Now().After(token.ExpiresAt) {
		return "", ErrInvalidToken
	}

	return token.UserID, nil
}

// RevokeToken invalidates a raw bearer token immediately, without deleting
// its record.
func (a *Authenticator) RevokeToken(ctx context.Context, rawToken string) error {
	token, err := a.Store.GetTokenByHash(ctx, hashToken(rawToken))
	if err != nil {
		return fmt.Errorf("looking up token: %w", err)
	}
	if token == nil {
		return ErrInvalidToken
	}

	return a.Store.RevokeToken(ctx, token.ID, time.Now())
}

// issueToken creates a new access token for userID and returns the raw
// token — the only time it's ever available in plaintext. Only its SHA-256
// hash is persisted, so a stolen index dump can't be replayed as a valid
// credential.
func (a *Authenticator) issueToken(ctx context.Context, userID string) (string, error) {
	raw, err := randomToken()
	if err != nil {
		return "", fmt.Errorf("generating token: %w", err)
	}

	now := time.Now()
	_, err = a.Store.CreateToken(ctx, model.Token{
		UserID:    userID,
		TokenHash: hashToken(raw),
		CreatedAt: now,
		ExpiresAt: now.Add(TokenTTL),
	})
	if err != nil {
		return "", fmt.Errorf("creating token: %w", err)
	}

	return raw, nil
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("reading random bytes: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(sum[:])
}
