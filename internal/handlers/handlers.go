package handlers

import (
	"net/http"

	"github.com/casapps/casdash/internal/app"
	"github.com/casapps/casdash/internal/websocket"
	"github.com/gorilla/sessions"
)

// Handlers contains all HTTP handlers
type Handlers struct {
	app          *app.App
	sessionStore *sessions.CookieStore
	wsHub        *websocket.Hub
}

// New creates a new handlers instance
func New(app *app.App, store *sessions.CookieStore, wsHub *websocket.Hub) *Handlers {
	return &Handlers{
		app:          app,
		sessionStore: store,
		wsHub:        wsHub,
	}
}

// Favicon serves the favicon
func (h *Handlers) Favicon(w http.ResponseWriter, r *http.Request) {
	// Serve a simple favicon
	w.Header().Set("Content-Type", "image/x-icon")
	w.WriteHeader(http.StatusOK)
}

// WebSocket Handlers
func (h *Handlers) WSStatus(w http.ResponseWriter, r *http.Request) {
	user := h.getCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	websocket.ServeWS(h.wsHub, w, r, user.ID)
}

func (h *Handlers) WSMonitoring(w http.ResponseWriter, r *http.Request) {
	user := h.getCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	websocket.ServeWS(h.wsHub, w, r, user.ID)
}

func (h *Handlers) WSNotifications(w http.ResponseWriter, r *http.Request) {
	user := h.getCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	websocket.ServeWS(h.wsHub, w, r, user.ID)
}

func (h *Handlers) WSLogs(w http.ResponseWriter, r *http.Request) {
	user := h.getCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	websocket.ServeWS(h.wsHub, w, r, user.ID)
}

func (h *Handlers) WSChat(w http.ResponseWriter, r *http.Request) {
	user := h.getCurrentUser(r)
	if user == nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	websocket.ServeWS(h.wsHub, w, r, user.ID)
}