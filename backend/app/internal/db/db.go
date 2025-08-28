package db

import (
    "context"
    "fmt"
    "log"
    "time"

    "github.com/jackc/pgx/v5"
    "github.com/markbates/goth"
)

// User struct matching your database schema
type User struct {
    ID            int
    Provider      string
    IDByProvider  string
    Name          string
    Nickname      string
    Email         string
    Location      string
    Description   string
    AccessToken   string
    RefreshToken  string
    ExpiresAt     time.Time
    PictureLink   string
    CreatedAt     time.Time
    UpdatedAt     time.Time
}

func GetOrCreateUser(conn *pgx.Conn, authUser goth.User) (*User, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()

    // Try to get existing user
    user, err := GetUserByProviderID(ctx, conn, authUser.Provider, authUser.UserID)
    if err != nil {
        if err == pgx.ErrNoRows {
            // User doesn't exist, create new one
            user, err = CreateUser(ctx, conn, authUser)
            if err != nil {
                return nil, fmt.Errorf("error creating user: %w", err)
            }
            log.Printf("New user created: %s (%s)", authUser.Name, authUser.Email)
            return user, nil
        }
        return nil, fmt.Errorf("error getting user: %w", err)
    }

    // User exists, check if any information needs updating
    updated, err := UpdateUserIfChanged(ctx, conn, user.ID, authUser)
    if err != nil {
        return nil, fmt.Errorf("error updating user: %w", err)
    }

    if updated {
        log.Printf("User information updated: %s (DB ID: %d)", authUser.Name, user.ID)
        // Get the updated user record
        user, err = GetUserByID(ctx, conn, user.ID)
        if err != nil {
            return nil, fmt.Errorf("error getting updated user: %w", err)
        }
    } else {
        log.Printf("User already exists with current information: %s (DB ID: %d)", user.Name, user.ID)
    }

    return user, nil
}

func GetUserByProviderID(ctx context.Context, conn *pgx.Conn, provider, idByProvider string) (*User, error) {
    var user User
    err := conn.QueryRow(ctx, `
        SELECT
            id, provider, id_by_provider, name, nickname, email, location,
            description, access_token, refresh_token, expires_at,
            picture_link, created_at, updated_at
        FROM users
        WHERE provider = $1 AND id_by_provider = $2`,
        provider, idByProvider,
    ).Scan(
        &user.ID, &user.Provider, &user.IDByProvider, &user.Name,
        &user.Nickname, &user.Email, &user.Location, &user.Description,
        &user.AccessToken, &user.RefreshToken, &user.ExpiresAt,
        &user.PictureLink, &user.CreatedAt, &user.UpdatedAt,
    )
    if err != nil {
        return nil, err
    }
    return &user, nil
}

func GetUserByID(ctx context.Context, conn *pgx.Conn, userID int) (*User, error) {
    var user User
    err := conn.QueryRow(ctx, `
        SELECT
            id, provider, id_by_provider, name, nickname, email, location,
            description, access_token, refresh_token, expires_at,
            picture_link, created_at, updated_at
        FROM users
        WHERE id = $1`,
        userID,
    ).Scan(
        &user.ID, &user.Provider, &user.IDByProvider, &user.Name,
        &user.Nickname, &user.Email, &user.Location, &user.Description,
        &user.AccessToken, &user.RefreshToken, &user.ExpiresAt,
        &user.PictureLink, &user.CreatedAt, &user.UpdatedAt,
    )
    if err != nil {
        return nil, err
    }
    return &user, nil
}

func CreateUser(ctx context.Context, conn *pgx.Conn, authUser goth.User) (*User, error) {
    var user User
    err := conn.QueryRow(ctx, `
        INSERT INTO users (
            provider, id_by_provider, name, nickname, email, location,
            description, access_token, refresh_token, expires_at, picture_link
        ) VALUES (
            $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
        )
        RETURNING
            id, provider, id_by_provider, name, nickname, email, location,
            description, access_token, refresh_token, expires_at,
            picture_link, created_at, updated_at`,
        authUser.Provider,
        authUser.UserID, // This maps to id_by_provider in the database
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
        &user.ID, &user.Provider, &user.IDByProvider, &user.Name,
        &user.Nickname, &user.Email, &user.Location, &user.Description,
        &user.AccessToken, &user.RefreshToken, &user.ExpiresAt,
        &user.PictureLink, &user.CreatedAt, &user.UpdatedAt,
    )
    if err != nil {
        return nil, fmt.Errorf("database insert error: %w", err)
    }
    return &user, nil
}

func UpdateUserIfChanged(ctx context.Context, conn *pgx.Conn, userID int, authUser goth.User) (bool, error) {
    result, err := conn.Exec(ctx, `
        UPDATE users SET
            name = $1,
            nickname = $2,
            email = $3,
            location = $4,
            description = $5,
            access_token = $6,
            refresh_token = $7,
            expires_at = $8,
            picture_link = $9,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = $10 AND (
            name IS DISTINCT FROM $1 OR
            nickname IS DISTINCT FROM $2 OR
            email IS DISTINCT FROM $3 OR
            location IS DISTINCT FROM $4 OR
            description IS DISTINCT FROM $5 OR
            access_token IS DISTINCT FROM $6 OR
            refresh_token IS DISTINCT FROM $7 OR
            expires_at IS DISTINCT FROM $8 OR
            picture_link IS DISTINCT FROM $9
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
