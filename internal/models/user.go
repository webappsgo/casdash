package models

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/casapps/casdash/internal/database"
	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system
type User struct {
	ID                     int       `json:"id"`
	Username               string    `json:"username"`
	Email                  string    `json:"email"`
	PasswordHash           string    `json:"-"` // Never expose password hash
	Role                   string    `json:"role"`
	IsPrimaryAdmin         bool      `json:"is_primary_admin"`
	OrganizationID         *int      `json:"organization_id,omitempty"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
	LastLogin              *time.Time `json:"last_login,omitempty"`
	TwoFASecret            string    `json:"-"` // Never expose 2FA secret
	TwoFAEnabled           bool      `json:"two_fa_enabled"`
	APIKey                 string    `json:"-"` // Never expose API key
	APIKeyCreated          *time.Time `json:"api_key_created,omitempty"`
	SessionToken           string    `json:"-"` // Never expose session token
	SessionExpires         *time.Time `json:"session_expires,omitempty"`
	Active                 bool      `json:"active"`
	EmailVerified          bool      `json:"email_verified"`
	PasswordResetToken     string    `json:"-"` // Never expose reset token
	PasswordResetExpires   *time.Time `json:"password_reset_expires,omitempty"`
	Preferences            string    `json:"preferences,omitempty"` // JSON
	Metadata               string    `json:"metadata,omitempty"`    // JSON
}

// UserManager handles user operations
type UserManager struct {
	db       *database.DB
	settings *Settings
}

// NewUserManager creates a new user manager
func NewUserManager(db *database.DB, settings *Settings) (*UserManager, error) {
	return &UserManager{
		db:       db,
		settings: settings,
	}, nil
}

// CreateUser creates a new user
func (um *UserManager) CreateUser(username, email, password, role string) (*User, error) {
	// Validate password strength
	if err := um.validatePassword(password); err != nil {
		return nil, fmt.Errorf("password validation failed: %w", err)
	}

	// Hash password
	cost, _ := um.settings.GetInt("password_bcrypt_cost")
	if cost == 0 {
		cost = 12 // Default cost
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Check if this is the first user (primary admin)
	var userCount int
	err = um.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&userCount)
	if err != nil {
		return nil, fmt.Errorf("failed to count users: %w", err)
	}

	isPrimaryAdmin := userCount == 0
	if isPrimaryAdmin {
		role = "primary_admin"
	}

	// Generate API key
	apiKey, err := um.generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// Insert user
	query := `INSERT INTO users (username, email, password_hash, role, is_primary_admin,
						api_key, api_key_created, active, email_verified, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	result, err := um.db.Exec(query, username, email, string(hashedPassword), role,
		isPrimaryAdmin, apiKey, now, true, isPrimaryAdmin, now, now)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	userID, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get user ID: %w", err)
	}

	user := &User{
		ID:             int(userID),
		Username:       username,
		Email:          email,
		Role:           role,
		IsPrimaryAdmin: isPrimaryAdmin,
		Active:         true,
		EmailVerified:  isPrimaryAdmin,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	logrus.WithFields(logrus.Fields{
		"user_id":          userID,
		"username":         username,
		"role":             role,
		"is_primary_admin": isPrimaryAdmin,
	}).Info("User created successfully")

	return user, nil
}

// GetUserByID retrieves a user by ID
func (um *UserManager) GetUserByID(id int) (*User, error) {
	query := `SELECT id, username, email, password_hash, role, is_primary_admin,
					 organization_id, created_at, updated_at, last_login, two_fa_enabled,
					 api_key_created, session_expires, active, email_verified,
					 password_reset_expires, preferences, metadata
			  FROM users WHERE id = ?`

	user := &User{}
	var orgID sql.NullInt64
	var lastLogin sql.NullTime
	var apiKeyCreated sql.NullTime
	var sessionExpires sql.NullTime
	var resetExpires sql.NullTime
	var preferences sql.NullString
	var metadata sql.NullString

	err := um.db.QueryRow(query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.Role, &user.IsPrimaryAdmin, &orgID, &user.CreatedAt,
		&user.UpdatedAt, &lastLogin, &user.TwoFAEnabled,
		&apiKeyCreated, &sessionExpires, &user.Active, &user.EmailVerified,
		&resetExpires, &preferences, &metadata,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %d", id)
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Handle nullable fields
	if orgID.Valid {
		orgIDInt := int(orgID.Int64)
		user.OrganizationID = &orgIDInt
	}
	if lastLogin.Valid {
		user.LastLogin = &lastLogin.Time
	}
	if apiKeyCreated.Valid {
		user.APIKeyCreated = &apiKeyCreated.Time
	}
	if sessionExpires.Valid {
		user.SessionExpires = &sessionExpires.Time
	}
	if resetExpires.Valid {
		user.PasswordResetExpires = &resetExpires.Time
	}
	if preferences.Valid {
		user.Preferences = preferences.String
	}
	if metadata.Valid {
		user.Metadata = metadata.String
	}

	return user, nil
}

// GetUserByUsername retrieves a user by username
func (um *UserManager) GetUserByUsername(username string) (*User, error) {
	query := `SELECT id FROM users WHERE username = ?`

	var userID int
	err := um.db.QueryRow(query, username).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", username)
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return um.GetUserByID(userID)
}

// GetUserByEmail retrieves a user by email
func (um *UserManager) GetUserByEmail(email string) (*User, error) {
	query := `SELECT id FROM users WHERE email = ?`

	var userID int
	err := um.db.QueryRow(query, email).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found: %s", email)
		}
		return nil, fmt.Errorf("failed to find user: %w", err)
	}

	return um.GetUserByID(userID)
}

// AuthenticateUser verifies username/password and returns user if valid
func (um *UserManager) AuthenticateUser(username, password string) (*User, error) {
	user, err := um.GetUserByUsername(username)
	if err != nil {
		return nil, err
	}

	if !user.Active {
		return nil, fmt.Errorf("user account is disabled")
	}

	// Check password
	err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password))
	if err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Update last login
	err = um.UpdateLastLogin(user.ID)
	if err != nil {
		logrus.WithError(err).Warn("Failed to update last login")
	}

	return user, nil
}

// UpdateLastLogin updates the user's last login time
func (um *UserManager) UpdateLastLogin(userID int) error {
	query := `UPDATE users SET last_login = ? WHERE id = ?`
	_, err := um.db.Exec(query, time.Now(), userID)
	return err
}

// UpdatePassword changes a user's password
func (um *UserManager) UpdatePassword(userID int, newPassword string) error {
	// Validate password strength
	if err := um.validatePassword(newPassword); err != nil {
		return fmt.Errorf("password validation failed: %w", err)
	}

	// Hash new password
	cost, _ := um.settings.GetInt("password_bcrypt_cost")
	if cost == 0 {
		cost = 12
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), cost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	query := `UPDATE users SET password_hash = ?, updated_at = ? WHERE id = ?`
	_, err = um.db.Exec(query, string(hashedPassword), time.Now(), userID)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	logrus.WithField("user_id", userID).Info("User password updated")
	return nil
}

// ListUsers returns all users (admin function)
func (um *UserManager) ListUsers() ([]*User, error) {
	query := `SELECT id, username, email, role, is_primary_admin, created_at,
					 last_login, active, email_verified
			  FROM users ORDER BY created_at DESC`

	rows, err := um.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		var lastLogin sql.NullTime

		err := rows.Scan(&user.ID, &user.Username, &user.Email, &user.Role,
			&user.IsPrimaryAdmin, &user.CreatedAt, &lastLogin, &user.Active,
			&user.EmailVerified)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}

		if lastLogin.Valid {
			user.LastLogin = &lastLogin.Time
		}

		users = append(users, user)
	}

	return users, nil
}

// validatePassword checks password strength
func (um *UserManager) validatePassword(password string) error {
	minLength, _ := um.settings.GetInt("password_min_length")
	if minLength == 0 {
		minLength = 12
	}

	if len(password) < minLength {
		return fmt.Errorf("password must be at least %d characters long", minLength)
	}

	// Add more password strength checks here
	return nil
}

// generateAPIKey generates a secure random API key
func (um *UserManager) generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// GetUserByAPIKey retrieves a user by their API key
func (um *UserManager) GetUserByAPIKey(apiKey string) (*User, error) {
	query := `SELECT id FROM users WHERE api_key = ? AND active = 1`

	var userID int
	err := um.db.QueryRow(query, apiKey).Scan(&userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid API key")
		}
		return nil, fmt.Errorf("failed to find user by API key: %w", err)
	}

	return um.GetUserByID(userID)
}

// CanUserAccessService checks if user can access a service based on mode
func (um *UserManager) CanUserAccessService(userID, serviceID int, mode string) (bool, error) {
	switch mode {
	case "enterprise":
		// In enterprise mode, check organization-based permissions
		return um.canAccessEnterpriseService(userID, serviceID)
	case "saas":
		// In SaaS mode, users can only access their own services
		return um.canAccessSaaSService(userID, serviceID)
	default:
		return false, fmt.Errorf("unknown mode: %s", mode)
	}
}

// canAccessEnterpriseService checks enterprise mode permissions
func (um *UserManager) canAccessEnterpriseService(userID, serviceID int) (bool, error) {
	// For now, all users in enterprise mode can access all services
	// TODO: Implement proper role-based permissions
	return true, nil
}

// canAccessSaaSService checks SaaS mode permissions
func (um *UserManager) canAccessSaaSService(userID, serviceID int) (bool, error) {
	query := `SELECT user_id FROM services WHERE id = ?`

	var serviceUserID sql.NullInt64
	err := um.db.QueryRow(query, serviceID).Scan(&serviceUserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("service not found")
		}
		return false, err
	}

	// Service must belong to the user
	return serviceUserID.Valid && int(serviceUserID.Int64) == userID, nil
}