package db

import (
	"github.com/jackc/pgx/v5"
	"time"
	"github.com/markbates/goth"
	"context"
	"log"
	"fmt"
)


// User struct matching your database schema
type User struct {
		ID					 int
		Provider		 string
		UserID			 string
		Name				 string
		Nickname		 string
		Email				string
		Location		 string
		Description	string
		AccessToken	string
		RefreshToken string
		ExpiresAt		time.Time
		PictureLink	string
		CreatedAt		time.Time
		UpdatedAt		time.Time
}

func GetOrCreateUser(conn *pgx.Conn, authUser goth.User) (*User, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		// Try to get existing user
		User, err := GetUserByProviderID(ctx, conn, authUser.Provider, authUser.UserID)
		if err != nil {
				if err == pgx.ErrNoRows {
						// User doesn't exist, create new one
						User, err = CreateUser(ctx, conn, authUser)
						if err != nil {
								return nil, fmt.Errorf("error creating user: %w", err)
						}
						log.Printf("New user created: %s (%s)", authUser.Name, authUser.Email)
						return User, nil
				}
				return nil, fmt.Errorf("error getting user: %w", err)
		}
		
		// User exists, check if any information needs updating
		updated, err := UpdateUserIfChanged(ctx, conn, User.ID, authUser)
		if err != nil {
				return nil, fmt.Errorf("error updating user: %w", err)
		}
		
		if updated {
				log.Printf("User information updated: %s (DB ID: %d)", authUser.Name, User.ID)
				// Get the updated user record
				User, err = GetUserByID(ctx, conn, User.ID)
				if err != nil {
						return nil, fmt.Errorf("error getting updated user: %w", err)
				}
		} else {
				log.Printf("User already exists with current information: %s (DB ID: %d)", User.Name, User.ID)
		}
		
		return User, nil
}

func GetUserByProviderID(ctx context.Context, conn *pgx.Conn, provider, userID string) (*User, error) {
		var User User
		err := conn.QueryRow(ctx, `
				SELECT 
						ID, PROVIDER, USERID, NAME, NICKNAME, EMAIL, LOCATION, 
						DESCRIPTION, ACCESSTOKEN, REFRESHTOKEN, EXPIRESAT, 
						PICTURELINK, CREATED_AT, UPDATED_AT
				FROM USERS 
				WHERE PROVIDER = $1 AND USERID = $2`,
				provider, userID,
		).Scan(
				&User.ID, &User.Provider, &User.UserID, &User.Name, 
				&User.Nickname, &User.Email, &User.Location, &User.Description,
				&User.AccessToken, &User.RefreshToken, &User.ExpiresAt,
				&User.PictureLink, &User.CreatedAt, &User.UpdatedAt,
		)
		
		if err != nil {
				return nil, err
		}
		
		return &User, nil
}

func GetUserByID(ctx context.Context, conn *pgx.Conn, userID int) (*User, error) {
		var User User
		err := conn.QueryRow(ctx, `
				SELECT 
						ID, PROVIDER, USERID, NAME, NICKNAME, EMAIL, LOCATION, 
						DESCRIPTION, ACCESSTOKEN, REFRESHTOKEN, EXPIRESAT, 
						PICTURELINK, CREATED_AT, UPDATED_AT
				FROM USERS 
				WHERE ID = $1`,
				userID,
		).Scan(
				&User.ID, &User.Provider, &User.UserID, &User.Name, 
				&User.Nickname, &User.Email, &User.Location, &User.Description,
				&User.AccessToken, &User.RefreshToken, &User.ExpiresAt,
				&User.PictureLink, &User.CreatedAt, &User.UpdatedAt,
		)
		
		if err != nil {
				return nil, err
		}
		
		return &User, nil
}

func CreateUser(ctx context.Context, conn *pgx.Conn, authUser goth.User) (*User, error) {
		var User User
		
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
				&User.ID, &User.Provider, &User.UserID, &User.Name, 
				&User.Nickname, &User.Email, &User.Location, &User.Description,
				&User.AccessToken, &User.RefreshToken, &User.ExpiresAt,
				&User.PictureLink, &User.CreatedAt, &User.UpdatedAt,
		)
		
		if err != nil {
				return nil, fmt.Errorf("database insert error: %w", err)
		}
		
		return &User, nil
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
