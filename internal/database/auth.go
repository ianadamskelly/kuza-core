package database

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var ErrUnauthorized = errors.New("unauthorized")

type User struct {
	ID          string    `json:"id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Membership struct {
	OrganizationID string `json:"organization_id"`
	Role           string `json:"role"`
}

type AuthUser struct {
	User        User         `json:"user"`
	Memberships []Membership `json:"memberships"`
}

type LoginParams struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type Session struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      AuthUser  `json:"user"`
}

func (db *DB) Login(ctx context.Context, input LoginParams, ttl time.Duration) (Session, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if email == "" || input.Password == "" {
		return Session{}, fmt.Errorf("%w: email and password are required", ErrInvalidInput)
	}
	if ttl <= 0 {
		return Session{}, fmt.Errorf("%w: session ttl must be positive", ErrInvalidInput)
	}

	var user User
	var passwordHash string
	if err := db.pool.QueryRow(ctx, `
		SELECT id, email, display_name, password_hash, created_at, updated_at
		FROM users
		WHERE email = $1
	`, email).Scan(
		&user.ID,
		&user.Email,
		&user.DisplayName,
		&passwordHash,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return Session{}, ErrUnauthorized
	}

	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(input.Password)); err != nil {
		return Session{}, ErrUnauthorized
	}

	token, err := randomToken()
	if err != nil {
		return Session{}, fmt.Errorf("generate session token: %w", err)
	}

	expiresAt := time.Now().UTC().Add(ttl)
	if _, err := db.pool.Exec(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, user.ID, tokenHash(token), expiresAt); err != nil {
		return Session{}, fmt.Errorf("create session: %w", err)
	}

	authUser, err := db.authUser(ctx, user)
	if err != nil {
		return Session{}, err
	}

	return Session{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      authUser,
	}, nil
}

func (db *DB) Authenticate(ctx context.Context, token string) (AuthUser, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return AuthUser{}, ErrUnauthorized
	}

	var user User
	if err := db.pool.QueryRow(ctx, `
		SELECT users.id, users.email, users.display_name, users.created_at, users.updated_at
		FROM sessions
		JOIN users ON users.id = sessions.user_id
		WHERE sessions.token_hash = $1
		  AND sessions.expires_at > now()
	`, tokenHash(token)).Scan(
		&user.ID,
		&user.Email,
		&user.DisplayName,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return AuthUser{}, ErrUnauthorized
	}

	return db.authUser(ctx, user)
}

func (db *DB) authUser(ctx context.Context, user User) (AuthUser, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT organization_id, role
		FROM memberships
		WHERE user_id = $1
		ORDER BY created_at ASC
	`, user.ID)
	if err != nil {
		return AuthUser{}, fmt.Errorf("query memberships: %w", err)
	}
	defer rows.Close()

	authUser := AuthUser{User: user, Memberships: []Membership{}}
	for rows.Next() {
		var membership Membership
		if err := rows.Scan(&membership.OrganizationID, &membership.Role); err != nil {
			return AuthUser{}, fmt.Errorf("scan membership: %w", err)
		}
		authUser.Memberships = append(authUser.Memberships, membership)
	}
	if err := rows.Err(); err != nil {
		return AuthUser{}, fmt.Errorf("iterate memberships: %w", err)
	}

	return authUser, nil
}

func (user AuthUser) HasRole(role string) bool {
	for _, membership := range user.Memberships {
		if membership.Role == role {
			return true
		}
	}
	return false
}

func randomToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func tokenHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
