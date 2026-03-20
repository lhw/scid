package api

import (
	"log/slog"
	"net/http"
)

type signupTokenResponse struct {
	Token string `json:"token"`
}

// handleSignupToken creates a single-use Pocket ID signup token and returns it
// so the frontend can construct the registration URL and redirect the user.
//
// POST /api/auth/signup-token  (public — no auth required)
//
// TODO: add rate limiting before public launch to prevent token farming.
func (s *Server) handleSignupToken(w http.ResponseWriter, r *http.Request) {
	st, err := s.pid.CreateSignupToken(r.Context())
	if err != nil {
		slog.ErrorContext(r.Context(), "create signup token failed", "err", err)
		writeError(w, http.StatusInternalServerError, "could not create signup token")
		return
	}

	writeJSON(w, http.StatusCreated, signupTokenResponse{Token: st.Token})
}
