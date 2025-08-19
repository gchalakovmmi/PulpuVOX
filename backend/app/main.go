package main

import (
	"os"
	"fmt"
	"log"
	"net/http"
	"PulpuVOX/pages/home"
	"PulpuVOX/pages/landing"
	"github.com/a-h/templ"
	"github.com/jackc/pgx/v5"
	"github.com/gchalakovmmi/PulpuWEB/db"

	"time"
	"github.com/gchalakovmmi/PulpuWEB/auth"
	"context"

	"github.com/markbates/goth"
)

func main() {
	// Get db connection details
	dbConnectionDetails, err := db.GetPostgresConfig()
	if err != nil {
			log.Fatalf("Failed to get Postgres config: %v", err)
	}

	// Initialize authentication
	authConfig, err := auth.GetGoogleAuthConfig()
	if err != nil {
		log.Fatalf("Error getting Google auth config: %v", err)
	}

	googleAuth := auth.NewGoogleAuth(authConfig)

	// Handle routes
	http.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "static/logo/favicon.ico")
	})
	http.HandleFunc("/", googleAuth.WithOutGoogleAuth("/home", func(w http.ResponseWriter, r *http.Request) {
		templ.Handler(landing.Landing()).ServeHTTP(w, r)
	}))
	// Authentication routes
	http.HandleFunc("/auth/google", func(w http.ResponseWriter, r *http.Request) {
		if _, err := googleAuth.GetSession(r); err == nil {
			http.Redirect(w, r, "/home", http.StatusSeeOther)
			return
		}
		googleAuth.BeginAuthHandler(w, r)
	})

	http.HandleFunc("/auth/google/callback", func(w http.ResponseWriter, r *http.Request) {
		user, err := googleAuth.CompleteUserAuth(w, r)
		if err != nil {
			http.Error(w, "Authentication failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := googleAuth.StoreSession(w, user); err != nil {
			http.Error(w, "Session creation failed: "+err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/home", http.StatusSeeOther)
	})

	http.HandleFunc("/logout/google", func(w http.ResponseWriter, r *http.Request) {
		googleAuth.LogoutHandler(w, r)
		googleAuth.ClearSession(w)
		http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
	})

	http.HandleFunc("/home", 
		googleAuth.WithGoogleAuth(
			db.WithDB(dbConnectionDetails, func(w http.ResponseWriter, r *http.Request, conn *pgx.Conn) {
				HomeHandler(w, r, conn, googleAuth)
			}),
		),
	)

	port := os.Getenv("BACKEND_PORT")
	fmt.Printf("Serving on port %s ...\n", port)
	http.ListenAndServe(fmt.Sprintf(":%s", port), nil)
}

func HomeHandler(w http.ResponseWriter, r *http.Request, conn *pgx.Conn, googleAuth *auth.GoogleAuth) {
    session, err := googleAuth.GetSession(r)
    if err != nil {
        http.Redirect(w, r, "/auth/google", http.StatusTemporaryRedirect)
        return
    }
    user := session.User
    
    // Check and create or update user if needed
    dbUser, err := GetOrCreateUser(conn, user)
    if err != nil {
        log.Printf("User management error: %v", err)
        http.Error(w, "Failed to process user data", http.StatusInternalServerError)
        return
    }
    
    log.Printf("User processed: %s (DB ID: %d)", dbUser.Name, dbUser.ID)
    
    // Render protected content using templ
    templ.Handler(home.Home(&user)).ServeHTTP(w, r)
}

// User struct matching your database schema
type DBUser struct {
    ID           int
    Provider     string
    UserID       string
    Name         string
    Nickname     string
    Email        string
    Location     string
    Description  string
    AccessToken  string
    RefreshToken string
    ExpiresAt    time.Time
    PictureLink  string
    CreatedAt    time.Time
    UpdatedAt    time.Time
}

func GetOrCreateUser(conn *pgx.Conn, authUser goth.User) (*DBUser, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    // Try to get existing user
    dbUser, err := GetUserByProviderID(ctx, conn, authUser.Provider, authUser.UserID)
    if err != nil {
        if err == pgx.ErrNoRows {
            // User doesn't exist, create new one
            dbUser, err = CreateUser(ctx, conn, authUser)
            if err != nil {
                return nil, fmt.Errorf("error creating user: %w", err)
            }
            log.Printf("New user created: %s (%s)", authUser.Name, authUser.Email)
            return dbUser, nil
        }
        return nil, fmt.Errorf("error getting user: %w", err)
    }
    
    // User exists, check if any information needs updating
    updated, err := UpdateUserIfChanged(ctx, conn, dbUser.ID, authUser)
    if err != nil {
        return nil, fmt.Errorf("error updating user: %w", err)
    }
    
    if updated {
        log.Printf("User information updated: %s (DB ID: %d)", authUser.Name, dbUser.ID)
        // Get the updated user record
        dbUser, err = GetUserByID(ctx, conn, dbUser.ID)
        if err != nil {
            return nil, fmt.Errorf("error getting updated user: %w", err)
        }
    } else {
        log.Printf("User already exists with current information: %s (DB ID: %d)", dbUser.Name, dbUser.ID)
    }
    
    return dbUser, nil
}

func GetUserByProviderID(ctx context.Context, conn *pgx.Conn, provider, userID string) (*DBUser, error) {
    var dbUser DBUser
    err := conn.QueryRow(ctx, `
        SELECT 
            ID, PROVIDER, USERID, NAME, NICKNAME, EMAIL, LOCATION, 
            DESCRIPTION, ACCESSTOKEN, REFRESHTOKEN, EXPIRESAT, 
            PICTURELINK, CREATED_AT, UPDATED_AT
        FROM USERS 
        WHERE PROVIDER = $1 AND USERID = $2`,
        provider, userID,
    ).Scan(
        &dbUser.ID, &dbUser.Provider, &dbUser.UserID, &dbUser.Name, 
        &dbUser.Nickname, &dbUser.Email, &dbUser.Location, &dbUser.Description,
        &dbUser.AccessToken, &dbUser.RefreshToken, &dbUser.ExpiresAt,
        &dbUser.PictureLink, &dbUser.CreatedAt, &dbUser.UpdatedAt,
    )
    
    if err != nil {
        return nil, err
    }
    
    return &dbUser, nil
}

func GetUserByID(ctx context.Context, conn *pgx.Conn, userID int) (*DBUser, error) {
    var dbUser DBUser
    err := conn.QueryRow(ctx, `
        SELECT 
            ID, PROVIDER, USERID, NAME, NICKNAME, EMAIL, LOCATION, 
            DESCRIPTION, ACCESSTOKEN, REFRESHTOKEN, EXPIRESAT, 
            PICTURELINK, CREATED_AT, UPDATED_AT
        FROM USERS 
        WHERE ID = $1`,
        userID,
    ).Scan(
        &dbUser.ID, &dbUser.Provider, &dbUser.UserID, &dbUser.Name, 
        &dbUser.Nickname, &dbUser.Email, &dbUser.Location, &dbUser.Description,
        &dbUser.AccessToken, &dbUser.RefreshToken, &dbUser.ExpiresAt,
        &dbUser.PictureLink, &dbUser.CreatedAt, &dbUser.UpdatedAt,
    )
    
    if err != nil {
        return nil, err
    }
    
    return &dbUser, nil
}

func CreateUser(ctx context.Context, conn *pgx.Conn, authUser goth.User) (*DBUser, error) {
    var dbUser DBUser
    
    err := conn.QueryRow(ctx, `
        INSERT INTO USERS (
            PROVIDER, USERID, NAME, NICKNAME, EMAIL, LOCATION, 
            DESCRIPTION, ACCESSTOKEN, REFRESHTOKEN, EXPIRESAT, PICTURELINK
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
        )
        RETURNING 
            ID, PROVIDER, USERID, NAME, NICKNAME, EMAIL, LOCATION, 
            DESCRIPTION, ACCESSTOKEN, REFRESHTOKEN, EXPIRESAT, 
            PICTURELINK, CREATED_AT, UPDATED_AT`,
        authUser.Provider,
        authUser.UserID,
        authUser.Name,
        authUser.NickName,
        authUser.Email,
        authUser.Location,
        authUser.Description,
        authUser.AccessToken,
        authUser.RefreshToken,
        authUser.ExpiresAt,
        authUser.AvatarURL,
    ).Scan(
        &dbUser.ID, &dbUser.Provider, &dbUser.UserID, &dbUser.Name, 
        &dbUser.Nickname, &dbUser.Email, &dbUser.Location, &dbUser.Description,
        &dbUser.AccessToken, &dbUser.RefreshToken, &dbUser.ExpiresAt,
        &dbUser.PictureLink, &dbUser.CreatedAt, &dbUser.UpdatedAt,
    )
    
    if err != nil {
        return nil, fmt.Errorf("database insert error: %w", err)
    }
    
    return &dbUser, nil
}

func UpdateUserIfChanged(ctx context.Context, conn *pgx.Conn, userID int, authUser goth.User) (bool, error) {
    result, err := conn.Exec(ctx, `
        UPDATE USERS SET
            NAME = $1,
            NICKNAME = $2,
            EMAIL = $3,
            LOCATION = $4,
            DESCRIPTION = $5,
            ACCESSTOKEN = $6,
            REFRESHTOKEN = $7,
            EXPIRESAT = $8,
            PICTURELINK = $9,
            UPDATED_AT = CURRENT_TIMESTAMP
        WHERE ID = $10 AND (
            NAME IS DISTINCT FROM $1 OR
            NICKNAME IS DISTINCT FROM $2 OR
            EMAIL IS DISTINCT FROM $3 OR
            LOCATION IS DISTINCT FROM $4 OR
            DESCRIPTION IS DISTINCT FROM $5 OR
            ACCESSTOKEN IS DISTINCT FROM $6 OR
            REFRESHTOKEN IS DISTINCT FROM $7 OR
            EXPIRESAT IS DISTINCT FROM $8 OR
            PICTURELINK IS DISTINCT FROM $9
        )`,
        authUser.Name,
        authUser.NickName,
        authUser.Email,
        authUser.Location,
        authUser.Description,
        authUser.AccessToken,
        authUser.RefreshToken,
        authUser.ExpiresAt,
        authUser.AvatarURL,
        userID,
    )
    
    if err != nil {
        return false, fmt.Errorf("database update error: %w", err)
    }
    
    rowsAffected := result.RowsAffected()
    return rowsAffected > 0, nil
}
