package server

import (
	"net/http"

	"github.com/farhan-hamzah/Akesa/backend/internal/access"
	"github.com/farhan-hamzah/Akesa/backend/internal/audit"
	"github.com/farhan-hamzah/Akesa/backend/internal/auth"
	"github.com/farhan-hamzah/Akesa/backend/internal/hospital"
	appmiddleware "github.com/farhan-hamzah/Akesa/backend/internal/middleware"
	"github.com/farhan-hamzah/Akesa/backend/internal/patient"
	"github.com/farhan-hamzah/Akesa/backend/internal/patientqr"
	"github.com/farhan-hamzah/Akesa/backend/internal/user"
)

const (
	RolePatient = "PATIENT"
	RoleStaff   = "HOSPITAL_STAFF"
	RoleAdmin   = "ADMIN"
)

type Dependencies struct {
	UserLoader auth.UserLoader

	UserHandler      *user.Handler
	PatientHandler   *patient.Handler
	HospitalHandler  *hospital.Handler
	AccessHandler    *access.Handler
	AuditHandler     *audit.Handler
	PatientQRHandler *patientqr.Handler
}

type Server struct {
	mux *http.ServeMux
}

func New(deps Dependencies) *Server {
	mux := http.NewServeMux()

	server := &Server{mux: mux}
	server.registerRoutes(deps)

	return server
}

func (s *Server) Handler() http.Handler {
	return appmiddleware.Chain(
		appmiddleware.Recover,
		appmiddleware.Logging,
	)(s.mux)
}

func (s *Server) registerRoutes(deps Dependencies) {
	s.mux.HandleFunc("GET /{$}", rootHandler)
	s.mux.HandleFunc("GET /health", healthHandler)
	s.mux.HandleFunc("GET /api/v1/audit/verify-chain", deps.AuditHandler.VerifyChain)

	s.mux.Handle("POST /api/v1/users/sync", auth.RequireAuth(http.HandlerFunc(deps.UserHandler.Sync)))
	s.mux.Handle("GET /api/v1/me", auth.RequireAuth(http.HandlerFunc(deps.UserHandler.Me)))

	s.mux.Handle("GET /api/v1/hospitals",
		s.chain(deps, http.HandlerFunc(deps.HospitalHandler.ListActive)),
	)

	s.mux.Handle("GET /api/v1/patient/profile",
		s.chain(deps, http.HandlerFunc(deps.PatientHandler.GetMyProfile), RolePatient),
	)
	s.mux.Handle("POST /api/v1/patient/profile",
		s.chain(deps, http.HandlerFunc(deps.PatientHandler.CreateProfile), RolePatient),
	)
	s.mux.Handle("PUT /api/v1/patient/profile",
		s.chain(deps, http.HandlerFunc(deps.PatientHandler.UpdateProfile), RolePatient),
	)
	s.mux.Handle("GET /api/v1/patient/access-requests",
		s.chain(deps, http.HandlerFunc(deps.AccessHandler.ListMine), RolePatient),
	)
	s.mux.Handle("POST /api/v1/patient/access-requests/{id}/approve",
		s.chain(deps, http.HandlerFunc(deps.AccessHandler.Approve), RolePatient),
	)
	s.mux.Handle("POST /api/v1/patient/access-requests/{id}/reject",
		s.chain(deps, http.HandlerFunc(deps.AccessHandler.Reject), RolePatient),
	)
	s.mux.Handle("POST /api/v1/patient/access-requests/{id}/revoke",
		s.chain(deps, http.HandlerFunc(deps.AccessHandler.Revoke), RolePatient),
	)
	s.mux.Handle("GET /api/v1/patient/history",
		s.chain(deps, http.HandlerFunc(deps.AuditHandler.MyProfileHistory), RolePatient),
	)

	s.mux.Handle("POST /api/v1/hospital/access-requests",
		s.chain(deps, http.HandlerFunc(deps.AccessHandler.Create), RoleStaff),
	)
	s.mux.Handle("GET /api/v1/hospital/access-requests",
		s.chain(deps, http.HandlerFunc(deps.AccessHandler.ListForMyHospital), RoleStaff),
	)
	s.mux.Handle("GET /api/v1/hospital/access-requests/{id}/data",
		s.chain(deps, http.HandlerFunc(deps.AccessHandler.FetchData), RoleStaff),
	)

	s.mux.Handle("POST /api/v1/admin/hospitals",
		s.chain(deps, http.HandlerFunc(deps.HospitalHandler.Create), RoleAdmin),
	)
	s.mux.Handle("GET /api/v1/admin/hospitals",
		s.chain(deps, http.HandlerFunc(deps.HospitalHandler.List), RoleAdmin),
	)
	s.mux.Handle("PATCH /api/v1/admin/hospitals/{id}/verify",
		s.chain(deps, http.HandlerFunc(deps.HospitalHandler.Verify), RoleAdmin),
	)
	s.mux.Handle("PATCH /api/v1/admin/hospitals/{id}/activate",
		s.chain(deps, http.HandlerFunc(deps.HospitalHandler.Activate), RoleAdmin),
	)
	s.mux.Handle("PATCH /api/v1/admin/hospitals/{id}/deactivate",
		s.chain(deps, http.HandlerFunc(deps.HospitalHandler.Deactivate), RoleAdmin),
	)
	s.mux.Handle("POST /api/v1/admin/hospitals/{id}/staff",
		s.chain(deps, http.HandlerFunc(deps.HospitalHandler.AddStaff), RoleAdmin),
	)
	s.mux.Handle("GET /api/v1/patient/qr",
		s.chain(deps, http.HandlerFunc(deps.PatientQRHandler.Get), RolePatient),
	)
	s.mux.Handle("POST /api/v1/patient/qr/rotate",
		s.chain(deps, http.HandlerFunc(deps.PatientQRHandler.Rotate), RolePatient),
	)
}

func (s *Server) chain(deps Dependencies, final http.Handler, roles ...string) http.Handler {
	mws := []appmiddleware.Middleware{
		auth.RequireAuth,
		auth.WithUser(deps.UserLoader),
	}
	if len(roles) > 0 {
		mws = append(mws, auth.RequireRole(roles...))
	}
	return appmiddleware.Chain(mws...)(final)
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"name":"Akesa API","status":"running","health":"/health"}`))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
