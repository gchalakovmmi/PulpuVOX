package authentication

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/markbates/goth"
	"github.com/markbates/goth/gothic"
	"github.com/markbates/goth/providers/google"
)

type Config struct {
	GoogleKey       string
	GoogleSecret    string
	CallbackURL     string
	SecretKey       []byte
	SessionDuration time.Duration
}

type Session struct {
	User      goth.User `json:"user"`
	ExpiresAt time.Time `json:"expires_at"`
}

type GoogleAuth struct {
	config       *Config
	providerName string
}

func NewGoogleAuth(config *Config) *GoogleAuth {
	if config.SessionDuration == 0 {
		config.SessionDuration = 24 * time.Hour
	}
	
	provider := google.New(
		config.GoogleKey,
		config.GoogleSecret,
		config.CallbackURL,
		"email", "profile",
	)
	goth.UseProviders(provider)
	
	return &GoogleAuth{
		config:       config,
		providerName: "google",
	}
}

func (ga *GoogleAuth) SetProviderContext(r *http.Request) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), gothic.ProviderParamKey, ga.providerName))
}

func (ga *GoogleAuth) BeginAuthHandler(w http.ResponseWriter, r *http.Request) {
	gothic.BeginAuthHandler(w, ga.SetProviderContext(r))
}

func (ga *GoogleAuth) CompleteUserAuth(w http.ResponseWriter, r *http.Request) (goth.User, error) {
	return gothic.CompleteUserAuth(w, ga.SetProviderContext(r))
}

func (ga *GoogleAuth) LogoutHandler(w http.ResponseWriter, r *http.Request) {
	gothic.Logout(w, ga.SetProviderContext(r))
}

func (ga *GoogleAuth) GetSession(r *http.Request) (*Session, error) {
	cookie, err := r.Cookie("pulpu_session")
	if err != nil {
		return nil, err
	}

	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		return nil, errors.New("invalid session format")
	}

	data, err := base64.URLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, err
	}

	signature, err := base64.URLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, err
	}

	if !validSignature(ga.config.SecretKey, data, signature) {
		return nil, errors.New("invalid session signature")
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, errors.New("session expired")
	}

	return &session, nil
}

func (ga *GoogleAuth) StoreSession(w http.ResponseWriter, user goth.User) error {
	session := Session{
		User:      user,
		ExpiresAt: time.Now().Add(ga.config.SessionDuration),
	}

	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	encodedData := base64.URLEncoding.EncodeToString(data)
	signature := createSignature(ga.config.SecretKey, data)
	encodedSig := base64.URLEncoding.EncodeToString(signature)

	cookie := &http.Cookie{
		Name:     "pulpu_session",
		Value:    fmt.Sprintf("%s.%s", encodedData, encodedSig),
		Path:     "/",
		Expires:  session.ExpiresAt,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}

	http.SetCookie(w, cookie)
	return nil
}

func (ga *GoogleAuth) ClearSession(w http.ResponseWriter) {
	cookie := &http.Cookie{
		Name:     "pulpu_session",
		Value:    "",
		Path:     "/",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
	http.SetCookie(w, cookie)
}

func createSignature(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

func validSignature(key, data, signature []byte) bool {
	expected := createSignature(key, data)
	return hmac.Equal(signature, expected)
}

func GenerateSecretKey() ([]byte, error) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		return nil, err
	}
	return key, nil
}
