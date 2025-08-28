package middleware

import (
    "net/http"

    "github.com/gchalakovmmi/PulpuWEB/auth"
    "github.com/gchalakovmmi/PulpuWEB/db"
    "github.com/jackc/pgx/v5"
)

// WithDBAndAuth wraps a handler with both database and authentication middleware
func WithDBAndAuth(
    dbConnectionDetails db.ConnectionDetails,
    googleAuth *auth.GoogleAuth,
    handler func(http.ResponseWriter, *http.Request, *pgx.Conn),
) http.HandlerFunc {
    // First wrap with DB, then with auth
    dbHandler := db.WithDB(dbConnectionDetails, handler)
    return googleAuth.WithGoogleAuth(func(w http.ResponseWriter, r *http.Request) {
        dbHandler(w, r)
    })
}
