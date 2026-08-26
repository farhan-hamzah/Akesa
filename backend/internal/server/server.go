package server

import (
	"net/http"

	"github.com/farhan-hamzah/Akesa/backend/internal/auth"
	"github.com/farhan-hamzah/Akesa/backend/internal/patient"
	"github.com/farhan-hamzah/Akesa/backend/internal/user"
)

type Server struct {
	mux *http.ServeMux
}

func New(
	userHandler *user.Handler,
	patientHandler *patient.Handler,
) *Server {
	mux := http.NewServeMux()

	server := &Server{
		mux: mux,
	}

	server.registerRoutes(
		userHandler,
		patientHandler,
	)

	return server
}

func (s *Server) Handler() http.Handler {
	return corsMiddleware(s.mux)
}

func (s *Server) registerRoutes(
	userHandler *user.Handler,
	patientHandler *patient.Handler,
) {
	s.mux.HandleFunc(
		"/health",
		healthHandler,
	)

	s.mux.Handle(
		"/api/v1/users/sync",
		auth.RequireAuth(
			http.HandlerFunc(userHandler.Sync),
		),
	)

	s.mux.Handle(
		"/api/v1/me",
		auth.RequireAuth(
			http.HandlerFunc(userHandler.Me),
		),
	)

	s.mux.Handle(
		"/api/v1/patients/me",
		auth.RequireAuth(
			http.HandlerFunc(patientHandler.Me),
		),
	)
}

func healthHandler(
	w http.ResponseWriter,
	r *http.Request,
) {
	w.Header().Set(
		"Content-Type",
		"application/json",
	)

	w.WriteHeader(http.StatusOK)

	_, _ = w.Write([]byte(`{
		"status": "ok"
	}`))
}
