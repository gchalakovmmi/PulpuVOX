package auth

import (
    "net/http"

    "PulpuVOX/internal/db"
    "github.com/gchalakovmmi/PulpuWEB/auth"
    "github.com/jackc/pgx/v5"
)

type AuthHandler struct {
    googleAuth *auth.GoogleAuth
}

func NewAuthHandler(googleAuth *auth.GoogleAuth) *AuthHandler {
    return &AuthHandler{
        googleAuth: googleAuth,
    }
}

func (h *AuthHandler) BeginAuthHandler(w http.ResponseWriter, r *http.Request) {
    h.googleAuth.BeginAuthHandler(w, r)
}

// Change to accept *pgx.Conn (pointer) instead of pgx.Conn
func (h *AuthHandler) AuthCallbackHandlerWithDB(w http.ResponseWriter, r *http.Request, conn *pgx.Conn) {
    user, err := h.googleAuth.CompleteUserAuth(w, r)
    if err != nil {
        http.Error(w, "Authentication failed: "+err.Error(), http.StatusInternalServerError)
        return
    }

    // Store or update user in database
    _, err = db.GetOrCreateUser(conn, user)
    if err != nil {
        http.Error(w, "Failed to process user data: "+err.Error(), http.StatusInternalServerError)
        return
    }

    // Store session
    if err := h.googleAuth.StoreSession(w, user); err != nil {
        http.Error(w, "Session creation failed: "+err.Error(), http.StatusInternalServerError)
        return
    }

    http.Redirect(w, r, "/home", http.StatusSeeOther)
}

func (h *AuthHandler) LogoutHandler(w http.ResponseWriter, r *http.Request) {
    h.googleAuth.LogoutHandler(w, r)
    h.googleAuth.ClearSession(w)
    http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}
