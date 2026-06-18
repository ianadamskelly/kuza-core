package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type CreateUserParams struct {
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Password    string `json:"password"`
}

type AddMembershipParams struct {
	UserID string `json:"user_id"`
	Role   string `json:"role"`
}

type OrganizationMember struct {
	UserID      string    `json:"user_id"`
	Email       string    `json:"email"`
	DisplayName string    `json:"display_name"`
	Role        string    `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

func (db *DB) ListUsers(ctx context.Context) ([]User, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT id, email, display_name, created_at, updated_at
		FROM users
		ORDER BY display_name ASC, email ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("query users: %w", err)
	}
	defer rows.Close()

	users := []User{}
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ID, &user.Email, &user.DisplayName, &user.CreatedAt, &user.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate users: %w", err)
	}

	return users, nil
}

func (db *DB) CreateUser(ctx context.Context, input CreateUserParams) (User, error) {
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	input.DisplayName = strings.TrimSpace(input.DisplayName)
	if input.Email == "" || input.DisplayName == "" || input.Password == "" {
		return User{}, fmt.Errorf("%w: email, display_name, and password are required", ErrInvalidInput)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}

	var user User
	if err := db.pool.QueryRow(ctx, `
		INSERT INTO users (email, display_name, password_hash)
		VALUES ($1, $2, $3)
		RETURNING id, email, display_name, created_at, updated_at
	`, input.Email, input.DisplayName, string(hash)).Scan(
		&user.ID,
		&user.Email,
		&user.DisplayName,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return User{}, fmt.Errorf("insert user: %w", err)
	}

	return user, nil
}

func (db *DB) ListOrganizationMembers(ctx context.Context, organizationID string) ([]OrganizationMember, error) {
	rows, err := db.pool.Query(ctx, `
		SELECT users.id, users.email, users.display_name, memberships.role, memberships.created_at
		FROM memberships
		JOIN users ON users.id = memberships.user_id
		WHERE memberships.organization_id = $1
		ORDER BY users.display_name ASC, users.email ASC, memberships.role ASC
	`, organizationID)
	if err != nil {
		return nil, fmt.Errorf("query organization members: %w", err)
	}
	defer rows.Close()

	members := []OrganizationMember{}
	for rows.Next() {
		var member OrganizationMember
		if err := rows.Scan(&member.UserID, &member.Email, &member.DisplayName, &member.Role, &member.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan organization member: %w", err)
		}
		members = append(members, member)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate organization members: %w", err)
	}

	return members, nil
}

func (db *DB) AddMembership(ctx context.Context, organizationID string, input AddMembershipParams) (OrganizationMember, error) {
	input.UserID = strings.TrimSpace(input.UserID)
	input.Role = strings.TrimSpace(input.Role)
	if organizationID == "" || input.UserID == "" || input.Role == "" {
		return OrganizationMember{}, fmt.Errorf("%w: organization id, user_id, and role are required", ErrInvalidInput)
	}
	if !validMembershipRole(input.Role) {
		return OrganizationMember{}, fmt.Errorf("%w: unsupported role", ErrInvalidInput)
	}

	var member OrganizationMember
	if err := db.pool.QueryRow(ctx, `
		WITH inserted AS (
			INSERT INTO memberships (organization_id, user_id, role)
			VALUES ($1, $2, $3)
			ON CONFLICT (organization_id, user_id, role) DO UPDATE
			SET role = EXCLUDED.role
			RETURNING user_id, role, created_at
		)
		SELECT users.id, users.email, users.display_name, inserted.role, inserted.created_at
		FROM inserted
		JOIN users ON users.id = inserted.user_id
	`, organizationID, input.UserID, input.Role).Scan(
		&member.UserID,
		&member.Email,
		&member.DisplayName,
		&member.Role,
		&member.CreatedAt,
	); err != nil {
		return OrganizationMember{}, fmt.Errorf("insert membership: %w", err)
	}

	return member, nil
}

func validMembershipRole(role string) bool {
	switch role {
	case "owner", "admin", "teacher", "guardian", "learner":
		return true
	default:
		return false
	}
}
