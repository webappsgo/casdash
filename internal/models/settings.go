package models

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/casapps/casdash/internal/database"
	"github.com/sirupsen/logrus"
)

// Settings provides access to application settings stored in the database
type Settings struct {
	db *database.DB
}

// NewSettings creates a new settings manager
func NewSettings(db *database.DB) (*Settings, error) {
	return &Settings{db: db}, nil
}

// Get retrieves a setting value by key
func (s *Settings) Get(key string) (string, error) {
	var value string
	query := `SELECT value FROM settings WHERE key = ?`

	err := s.db.QueryRow(query, key).Scan(&value)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("setting not found: %s", key)
		}
		return "", fmt.Errorf("failed to get setting %s: %w", key, err)
	}

	return value, nil
}

// Set updates or creates a setting
func (s *Settings) Set(key, value string) error {
	return s.SetWithType(key, value, "string", "system", "")
}

// SetWithType updates or creates a setting with type and metadata
func (s *Settings) SetWithType(key, value, settingType, category, description string) error {
	query := `INSERT OR REPLACE INTO settings (key, value, type, category, description, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?)`

	_, err := s.db.Exec(query, key, value, settingType, category, description, time.Now())
	if err != nil {
		return fmt.Errorf("failed to set setting %s: %w", key, err)
	}

	logrus.WithFields(logrus.Fields{
		"key":   key,
		"type":  settingType,
		"category": category,
	}).Debug("Setting updated")

	return nil
}

// SetDefault sets a setting only if it doesn't already exist
func (s *Settings) SetDefault(key, value string) error {
	// Check if setting exists
	_, err := s.Get(key)
	if err == nil {
		// Setting exists, don't override
		return nil
	}

	// Setting doesn't exist, set it
	return s.Set(key, value)
}

// GetBool retrieves a boolean setting
func (s *Settings) GetBool(key string) (bool, error) {
	value, err := s.Get(key)
	if err != nil {
		return false, err
	}

	return strconv.ParseBool(value)
}

// SetBool sets a boolean setting
func (s *Settings) SetBool(key string, value bool) error {
	return s.SetWithType(key, strconv.FormatBool(value), "boolean", "system", "")
}

// GetInt retrieves an integer setting
func (s *Settings) GetInt(key string) (int, error) {
	value, err := s.Get(key)
	if err != nil {
		return 0, err
	}

	return strconv.Atoi(value)
}

// SetInt sets an integer setting
func (s *Settings) SetInt(key string, value int) error {
	return s.SetWithType(key, strconv.Itoa(value), "integer", "system", "")
}

// GetFloat retrieves a float setting
func (s *Settings) GetFloat(key string) (float64, error) {
	value, err := s.Get(key)
	if err != nil {
		return 0, err
	}

	return strconv.ParseFloat(value, 64)
}

// SetFloat sets a float setting
func (s *Settings) SetFloat(key string, value float64) error {
	return s.SetWithType(key, strconv.FormatFloat(value, 'f', -1, 64), "float", "system", "")
}

// GetByCategory retrieves all settings in a category
func (s *Settings) GetByCategory(category string) (map[string]string, error) {
	query := `SELECT key, value FROM settings WHERE category = ? ORDER BY key`

	rows, err := s.db.Query(query, category)
	if err != nil {
		return nil, fmt.Errorf("failed to get settings by category %s: %w", category, err)
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}
		settings[key] = value
	}

	return settings, nil
}

// GetAll retrieves all settings
func (s *Settings) GetAll() (map[string]Setting, error) {
	query := `SELECT key, value, type, category, description, updated_at FROM settings ORDER BY category, key`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get all settings: %w", err)
	}
	defer rows.Close()

	settings := make(map[string]Setting)
	for rows.Next() {
		var setting Setting
		if err := rows.Scan(&setting.Key, &setting.Value, &setting.Type,
			&setting.Category, &setting.Description, &setting.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan setting: %w", err)
		}
		settings[setting.Key] = setting
	}

	return settings, nil
}

// Delete removes a setting
func (s *Settings) Delete(key string) error {
	query := `DELETE FROM settings WHERE key = ?`

	result, err := s.db.Exec(query, key)
	if err != nil {
		return fmt.Errorf("failed to delete setting %s: %w", key, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("setting not found: %s", key)
	}

	logrus.WithField("key", key).Debug("Setting deleted")
	return nil
}

// Exists checks if a setting exists
func (s *Settings) Exists(key string) bool {
	_, err := s.Get(key)
	return err == nil
}

// Setting represents a setting with metadata
type Setting struct {
	Key         string    `json:"key"`
	Value       string    `json:"value"`
	Type        string    `json:"type"`
	Category    string    `json:"category"`
	Description string    `json:"description"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GetSystemSettings returns all system-category settings
func (s *Settings) GetSystemSettings() (map[string]string, error) {
	return s.GetByCategory("system")
}

// GetMonitoringSettings returns all monitoring-category settings
func (s *Settings) GetMonitoringSettings() (map[string]string, error) {
	return s.GetByCategory("monitoring")
}

// GetSecuritySettings returns all security-category settings
func (s *Settings) GetSecuritySettings() (map[string]string, error) {
	return s.GetByCategory("security")
}

// GetDiscoverySettings returns all discovery-category settings
func (s *Settings) GetDiscoverySettings() (map[string]string, error) {
	return s.GetByCategory("discovery")
}

// GetUISettings returns all ui-category settings
func (s *Settings) GetUISettings() (map[string]string, error) {
	return s.GetByCategory("ui")
}

// BulkSet sets multiple settings in a transaction
func (s *Settings) BulkSet(settings map[string]string) error {
	tx, err := s.db.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `INSERT OR REPLACE INTO settings (key, value, type, category, description, updated_at)
			  VALUES (?, ?, 'string', 'system', '', ?)`

	for key, value := range settings {
		_, err := tx.Exec(query, key, value, time.Now())
		if err != nil {
			return fmt.Errorf("failed to set setting %s: %w", key, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	logrus.WithField("count", len(settings)).Debug("Bulk settings update completed")
	return nil
}