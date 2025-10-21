package handlers

import (
	"encoding/json"
	"html/template"
	"net/http"

	"github.com/casapps/casdash/internal/models"
	"github.com/sirupsen/logrus"
)

// LoginData represents the login page data
type LoginData struct {
	BaseData
	Error        string
	FirstRun     bool
	Registration string
}

// Login handles user login
func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	// Check if first run
	firstRun := h.app.IsFirstRun()

	if r.Method == "GET" {
		// Display login form
		data := LoginData{
			BaseData: h.getBaseData(r, nil),
			FirstRun: firstRun,
		}

		// If first run, redirect to setup
		if firstRun {
			http.Redirect(w, r, "/setup", http.StatusFound)
			return
		}

		h.renderTemplate(w, "login.html", data)
		return
	}

	// Handle POST - process login
	if err := r.ParseForm(); err != nil {
		h.renderError(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	password := r.FormValue("password")

	if username == "" || password == "" {
		data := LoginData{
			BaseData: h.getBaseData(r, nil),
			Error:    "Username and password are required",
			FirstRun: firstRun,
		}
		h.renderTemplate(w, "login.html", data)
		return
	}

	// Authenticate user
	user, err := h.app.Users.AuthenticateUser(username, password)
	if err != nil {
		logrus.WithError(err).WithField("username", username).Warn("Login failed")
		data := LoginData{
			BaseData: h.getBaseData(r, nil),
			Error:    "Invalid username or password",
			FirstRun: firstRun,
		}
		h.renderTemplate(w, "login.html", data)
		return
	}

	// Create session
	session, err := h.sessionStore.Get(r, "casdash-session")
	if err != nil {
		h.renderError(w, "Session error", http.StatusInternalServerError)
		return
	}

	session.Values["user_id"] = user.ID
	session.Values["username"] = user.Username
	session.Values["role"] = user.Role
	session.Values["authenticated"] = true

	if err := session.Save(r, w); err != nil {
		logrus.WithError(err).Error("Failed to save session")
		h.renderError(w, "Failed to create session", http.StatusInternalServerError)
		return
	}

	logrus.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
		"role":     user.Role,
	}).Info("User logged in successfully")

	// Redirect to dashboard
	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

// Logout handles user logout
func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	session, err := h.sessionStore.Get(r, "casdash-session")
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return
	}

	// Get username for logging
	username, _ := session.Values["username"].(string)

	// Clear session
	session.Values = make(map[interface{}]interface{})
	session.Options.MaxAge = -1

	if err := session.Save(r, w); err != nil {
		logrus.WithError(err).Error("Failed to clear session")
	}

	logrus.WithField("username", username).Info("User logged out")

	http.Redirect(w, r, "/login", http.StatusFound)
}

// Register handles user registration
func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	// Check if registration is enabled
	registrationPolicy, _ := h.app.Settings.Get("registration")
	if registrationPolicy == "disabled" {
		h.renderError(w, "Registration is disabled", http.StatusForbidden)
		return
	}

	if r.Method == "GET" {
		data := h.getBaseData(r, nil)
		h.renderTemplate(w, "register.html", data)
		return
	}

	// Handle POST - process registration
	if err := r.ParseForm(); err != nil {
		h.renderError(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	// Validate inputs
	if username == "" || email == "" || password == "" {
		h.renderTemplateWithError(w, "register.html", "All fields are required")
		return
	}

	if password != confirmPassword {
		h.renderTemplateWithError(w, "register.html", "Passwords do not match")
		return
	}

	// Create user
	user, err := h.app.Users.CreateUser(username, email, password, "user")
	if err != nil {
		logrus.WithError(err).Error("Failed to create user")
		h.renderTemplateWithError(w, "register.html", "Failed to create account: "+err.Error())
		return
	}

	logrus.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
		"email":    user.Email,
	}).Info("New user registered")

	// Auto-login after registration
	session, _ := h.sessionStore.Get(r, "casdash-session")
	session.Values["user_id"] = user.ID
	session.Values["username"] = user.Username
	session.Values["role"] = user.Role
	session.Values["authenticated"] = true
	session.Save(r, w)

	http.Redirect(w, r, "/dashboard", http.StatusFound)
}

// Setup handles first-run setup wizard
func (h *Handlers) Setup(w http.ResponseWriter, r *http.Request) {
	// Only allow setup on first run
	if !h.app.IsFirstRun() {
		http.Redirect(w, r, "/dashboard", http.StatusFound)
		return
	}

	if r.Method == "GET" {
		data := struct {
			BaseData
			Mode string
		}{
			BaseData: h.getBaseData(r, nil),
			Mode:     h.app.Mode,
		}
		h.renderTemplate(w, "setup.html", data)
		return
	}

	// Handle POST - create primary admin
	if err := r.ParseForm(); err != nil {
		h.renderError(w, "Invalid form data", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	// Validate
	if username == "" || email == "" || password == "" {
		h.renderTemplateWithError(w, "setup.html", "All fields are required")
		return
	}

	if password != confirmPassword {
		h.renderTemplateWithError(w, "setup.html", "Passwords do not match")
		return
	}

	// Create primary admin user
	user, err := h.app.Users.CreateUser(username, email, password, "primary_admin")
	if err != nil {
		logrus.WithError(err).Error("Failed to create primary admin")
		h.renderTemplateWithError(w, "setup.html", "Failed to create admin account: "+err.Error())
		return
	}

	// Mark setup as complete
	h.app.Settings.Set("first_run", "false")

	logrus.WithFields(logrus.Fields{
		"user_id":  user.ID,
		"username": user.Username,
	}).Info("Primary admin created, setup complete")

	// Auto-login
	session, _ := h.sessionStore.Get(r, "casdash-session")
	session.Values["user_id"] = user.ID
	session.Values["username"] = user.Username
	session.Values["role"] = user.Role
	session.Values["authenticated"] = true
	session.Save(r, w)

	http.Redirect(w, r, "/dashboard", http.StatusFound)
}


// APILogout handles API logout
func (h *Handlers) APILogout(w http.ResponseWriter, r *http.Request) {
	session, _ := h.sessionStore.Get(r, "casdash-session")
	session.Values = make(map[interface{}]interface{})
	session.Options.MaxAge = -1
	session.Save(r, w)

	h.jsonResponse(w, map[string]interface{}{
		"success": true,
		"message": "Logout successful",
	})
}

// APIProfile returns the current user's profile
func (h *Handlers) APIProfile(w http.ResponseWriter, r *http.Request) {
	user := h.getCurrentUser(r)
	if user == nil {
		h.jsonError(w, "Not authenticated", http.StatusUnauthorized)
		return
	}

	h.jsonResponse(w, map[string]interface{}{
		"id":            user.ID,
		"username":      user.Username,
		"email":         user.Email,
		"role":          user.Role,
		"created_at":    user.CreatedAt,
		"last_login":    user.LastLogin,
		"email_verified": user.EmailVerified,
	})
}

// Helper functions

// getCurrentUser retrieves the current authenticated user from session
func (h *Handlers) getCurrentUser(r *http.Request) *models.User {
	session, err := h.sessionStore.Get(r, "casdash-session")
	if err != nil {
		return nil
	}

	authenticated, ok := session.Values["authenticated"].(bool)
	if !ok || !authenticated {
		return nil
	}

	userID, ok := session.Values["user_id"].(int)
	if !ok {
		return nil
	}

	user, err := h.app.Users.GetUserByID(userID)
	if err != nil {
		return nil
	}

	return user
}

// isAuthenticated checks if the current request is authenticated
func (h *Handlers) isAuthenticated(r *http.Request) bool {
	return h.getCurrentUser(r) != nil
}

// requireAuth is a helper to check authentication
func (h *Handlers) requireAuth(w http.ResponseWriter, r *http.Request) *models.User {
	user := h.getCurrentUser(r)
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusFound)
		return nil
	}
	return user
}

// requireRole checks if user has the required role
func (h *Handlers) requireRole(w http.ResponseWriter, r *http.Request, requiredRole string) *models.User {
	user := h.requireAuth(w, r)
	if user == nil {
		return nil
	}

	// Check role hierarchy
	roleHierarchy := map[string]int{
		"primary_admin": 100,
		"admin":         80,
		"support":       60,
		"user":          40,
		"view_only":     20,
	}

	userLevel := roleHierarchy[user.Role]
	requiredLevel := roleHierarchy[requiredRole]

	if userLevel < requiredLevel {
		h.renderError(w, "Insufficient permissions", http.StatusForbidden)
		return nil
	}

	return user
}

// getBaseData returns base template data
func (h *Handlers) getBaseData(r *http.Request, user *models.User) BaseData {
	if user == nil {
		user = h.getCurrentUser(r)
	}

	version := h.app.GetVersion()

	return BaseData{
		User:          user,
		Mode:          h.app.Mode,
		Version:       version["version"],
		ExecutionTime: 0, // Will be calculated by template
	}
}

// renderTemplate renders an HTML template
func (h *Handlers) renderTemplate(w http.ResponseWriter, templateName string, data interface{}) {
	tmpl, err := template.ParseFiles(
		"internal/server/templates/base.html",
		"internal/server/templates/"+templateName,
	)
	if err != nil {
		logrus.WithError(err).Error("Failed to parse template")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(w, "base.html", data); err != nil {
		logrus.WithError(err).Error("Failed to execute template")
	}
}

// renderTemplateWithError renders a template with an error message
func (h *Handlers) renderTemplateWithError(w http.ResponseWriter, templateName string, errorMsg string) {
	data := struct {
		BaseData
		Error string
	}{
		BaseData: h.getBaseData(nil, nil),
		Error:    errorMsg,
	}
	h.renderTemplate(w, templateName, data)
}

// renderError renders a generic error page
func (h *Handlers) renderError(w http.ResponseWriter, message string, statusCode int) {
	w.WriteHeader(statusCode)
	data := struct {
		BaseData
		Error      string
		StatusCode int
	}{
		BaseData:   h.getBaseData(nil, nil),
		Error:      message,
		StatusCode: statusCode,
	}
	h.renderTemplate(w, "error.html", data)
}

// jsonResponse sends a JSON response
func (h *Handlers) jsonResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// jsonError sends a JSON error response
func (h *Handlers) jsonError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error":   message,
		"success": false,
	})
}

// BaseData represents common template data
type BaseData struct {
	User          *models.User
	Mode          string
	Version       string
	ExecutionTime int64
	ActivePage    string
	Flash         []FlashMessage
}

// FlashMessage represents a flash message
type FlashMessage struct {
	Type    string
	Message string
}

// addFlash adds a flash message to the session
func (h *Handlers) addFlash(r *http.Request, w http.ResponseWriter, msgType, message string) {
	session, _ := h.sessionStore.Get(r, "casdash-session")

	flashes := []FlashMessage{}
	if existing, ok := session.Values["flashes"].([]FlashMessage); ok {
		flashes = existing
	}

	flashes = append(flashes, FlashMessage{
		Type:    msgType,
		Message: message,
	})

	session.Values["flashes"] = flashes
	session.Save(r, w)
}

// getFlashes retrieves and clears flash messages
func (h *Handlers) getFlashes(r *http.Request, w http.ResponseWriter) []FlashMessage {
	session, _ := h.sessionStore.Get(r, "casdash-session")

	flashes := []FlashMessage{}
	if existing, ok := session.Values["flashes"].([]FlashMessage); ok {
		flashes = existing
	}

	// Clear flashes
	session.Values["flashes"] = []FlashMessage{}
	session.Save(r, w)

	return flashes
}