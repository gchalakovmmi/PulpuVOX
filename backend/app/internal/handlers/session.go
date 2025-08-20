package handlers

import (
    "net/http"
    "context"
    "github.com/jackc/pgx/v5"
    "github.com/gchalakovmmi/PulpuWEB/auth"
    "PulpuVOX/internal/db"
)

func getUserIdFromSession(r *http.Request, conn *pgx.Conn, googleAuth *auth.GoogleAuth) (int, error) {
    session, err := googleAuth.GetSession(r)
    if err != nil {
        return 0, err
    }
    
    user := session.User
    dbUser, err := db.GetUserByProviderID(context.Background(), conn, user.Provider, user.UserID)
    if err != nil {
        return 0, err
    }
    
    return dbUser.ID, nil
}
