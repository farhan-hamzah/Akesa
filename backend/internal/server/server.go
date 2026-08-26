package server

import (
	"net/http"

	"github.com/farhan-hamzah/Akesa/backend/internal/access"
	"github.com/farhan-hamzah/Akesa/backend/internal/audit"
	"github.com/farhan-hamzah/Akesa/backend/internal/auth"
	"github.com/farhan-hamzah/Akesa/backend/internal/hospital"
	appmiddleware "github.com/farhan-hamzah/Akesa/backend/internal/middleware"
	"github.com/farhan-hamzah/Akesa/backend/internal/patient"
	"github.com/farhan-hamzah/Akesa/backend/internal/user"
)

// Roles - keep string literals in one place so routing and the `user`
// package's Role type can never silently drift apart.
const (
	RolePatient = "PATIENT"
	RoleStaff   = "HOSPITAL_STAFF"
	RoleAdmin   = "ADMIN"
)

// Dependencies holds every handler and the auth.UserLoader needed to wire
// routes. Passing one struct instead of a long positional argument list
// keeps main.go readable as more domains get added.
type Dependencies struct {
	UserLoader auth.UserLoader // typically *user.Service

	UserHandler     *user.Handler
	PatientHandler  *patient.Handler
	HospitalHandler *hospital.Handler
	AccessHandler   *access.Handler
	AuditHandler    *audit.Handler
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

// Handler returns the fully wrapped root handler (recovery + logging on
// top of routing) - this is what main.go passes to http.ListenAndServe.
func (s *Server) Handler() http.Handler {
	return appmiddleware.Chain(
		appmiddleware.Recover,
		appmiddleware.Logging,
	)(s.mux)
}

func (s *Server) registerRoutes(deps Dependencies) {
	s.mux.HandleFunc("GET /health", healthHandler)

	// --- Account bootstrap: any authenticated Clerk principal, no role
	// check yet (role doesn't exist in our DB until /sync runs once). ---
	s.mux.Handle("POST /api/v1/users/sync", auth.RequireAuth(http.HandlerFunc(deps.UserHandler.Sync)))
	s.mux.Handle("GET /api/v1/me", auth.RequireAuth(http.HandlerFunc(deps.UserHandler.Me)))

	// Shared "browse active hospitals" - any synced, active account.
	s.mux.Handle("GET /api/v1/hospitals",
		s.chain(deps, http.HandlerFunc(deps.HospitalHandler.ListActive)),
	)

	// --- Patient routes ---
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

	// --- Hospital staff routes ---
	s.mux.Handle("POST /api/v1/hospital/access-requests",
		s.chain(deps, http.HandlerFunc(deps.AccessHandler.Create), RoleStaff),
	)
	s.mux.Handle("GET /api/v1/hospital/access-requests",
		s.chain(deps, http.HandlerFunc(deps.AccessHandler.ListForMyHospital), RoleStaff),
	)
	s.mux.Handle("GET /api/v1/hospital/access-requests/{id}/data",
		s.chain(deps, http.HandlerFunc(deps.AccessHandler.FetchData), RoleStaff),
	)

	// --- Admin routes ---
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
}

// chain wraps a handler with RequireAuth -> WithUser -> (optionally)
// RequireRole, so every protected route is built the same, predictable
// way. Pass no roles to require only "authenticated + synced", any roles
// to restrict further.
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

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ok"}`))
}
