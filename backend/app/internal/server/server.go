package server

import (
		"context"
		"net/http"

	 "PulpuVOX/internal/middleware"
	 "github.com/jackc/pgx/v5"
		"PulpuVOX/internal/config"
		appAuth "PulpuVOX/internal/handlers/auth"
		"PulpuVOX/internal/handlers/feedback"
		"PulpuVOX/internal/handlers/conversation"
		"PulpuVOX/internal/handlers/conversationanalysis"
		"PulpuVOX/internal/handlers/health"
		"PulpuVOX/internal/handlers/home"
		"PulpuVOX/internal/handlers/landing"
		"PulpuVOX/internal/services"
		pulpuwebAuth "github.com/gchalakovmmi/PulpuWEB/auth"
		"github.com/gchalakovmmi/PulpuWEB/db"
)

type Server struct {
		config							config.Config
		googleAuth					*pulpuwebAuth.GoogleAuth
		authHandler				 *appAuth.AuthHandler
		dbConnectionDetails db.ConnectionDetails
		services						*services.Services
}

func (s *Server) withDBAndAuth(handler func(http.ResponseWriter, *http.Request, *pgx.Conn)) http.HandlerFunc {
		return s.googleAuth.WithGoogleAuth(func(w http.ResponseWriter, r *http.Request) {
				// We'll use the db.WithDB middleware internally
				db.WithDB(s.dbConnectionDetails, handler)(w, r)
		})
}

func New(cfg config.Config) *Server {
		// Initialize Google authentication
		authConfig, err := pulpuwebAuth.GetGoogleAuthConfig()
		if err != nil {
				panic("Failed to get Google auth config: " + err.Error())
		}
		
		googleAuth := pulpuwebAuth.NewGoogleAuth(authConfig)
		authHandler := appAuth.NewAuthHandler(googleAuth)

		// Initialize database connection details
		dbConnectionDetails, err := db.GetPostgresConfig()
		if err != nil {
				panic("Failed to get database config: " + err.Error())
		}

		// Initialize services
		services := services.New()

		return &Server{
				config:							cfg,
				googleAuth:					googleAuth,
				authHandler:				 authHandler,
				dbConnectionDetails: dbConnectionDetails,
				services:						services,
		}
}

func (s *Server) createHandler() http.Handler {
    mux := http.NewServeMux()
    
    // Serve static files
    mux.Handle("/static/", http.StripPrefix("/static/",
        http.FileServer(http.Dir("./web/static"))))
    
    // Health check endpoint
    mux.HandleFunc("/health", health.Handler)
    
    // Authentication routes
    mux.HandleFunc("/auth/google", s.authHandler.BeginAuthHandler)
    mux.HandleFunc("/auth/google/callback",
        db.WithDB(s.dbConnectionDetails, s.authHandler.AuthCallbackHandlerWithDB))
    mux.HandleFunc("/logout/google", s.authHandler.LogoutHandler)
    
    // Application routes with authentication middleware
    mux.Handle("/", s.googleAuth.WithOutGoogleAuth("/home", landing.Handler))
    mux.Handle("/home", s.withUserContext(s.googleAuth.WithGoogleAuth(home.Handler)))
    mux.Handle("/conversation", s.withUserContext(s.googleAuth.WithGoogleAuth(conversation.Handler)))
    mux.Handle("/conversation-analysis", s.withUserContext(s.googleAuth.WithGoogleAuth(conversationanalysis.Handler)))
    
    // API routes
    mux.Handle("/api/conversation/turn",
        s.withUserContext(
            middleware.WithDBAndAuth(s.dbConnectionDetails, s.googleAuth,
                func(w http.ResponseWriter, r *http.Request, conn *pgx.Conn) {
                    conversation.APIConversationHandler(s.services.WhisperService, s.services.OpenAIClient, s.services.TTSService)(w, r, conn)
                }),
        ),
    )
    
    // Add conversation end handler - only register this once
    mux.Handle("/api/conversation/end",
        middleware.WithDBAndAuth(s.dbConnectionDetails, s.googleAuth, conversation.ConversationEndHandler))
    
    // Add conversation analysis API endpoint
    mux.Handle("/api/conversation/latest",
        middleware.WithDBAndAuth(s.dbConnectionDetails, s.googleAuth, conversationanalysis.GetLatestConversationHandler))
    
    // Add feedback generation endpoint
    mux.Handle("/api/feedback/generate",
        middleware.WithDBAndAuth(s.dbConnectionDetails, s.googleAuth, feedback.GenerateFeedbackHandler))
    
    return mux
}

// Middleware to add user to context
func (s *Server) withUserContext(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
				session, err := s.googleAuth.GetSession(r)
				if err == nil && session != nil {
						// Add user to context
						ctx := context.WithValue(r.Context(), "user", &session.User)
						r = r.WithContext(ctx)
				}
				next(w, r)
		}
}

func (s *Server) ListenAndServe() error {
		server := &http.Server{
				Addr:				 ":" + s.config.Port,
				Handler:			s.createHandler(),
				ReadTimeout:	s.config.ReadTimeout,
				WriteTimeout: s.config.WriteTimeout,
				IdleTimeout:	s.config.IdleTimeout,
		}
		
		return server.ListenAndServe()
}
