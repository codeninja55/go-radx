package server

import (
	"context"
	"net/http"

	"go.uber.org/zap"
)

// principalKey is the unexported context key the auth middleware attaches the authenticated
// Principal under, so a downstream handler reads it without a package-global and without colliding
// with another package's context values.
type principalKey struct{}

// unauthorizedResponder writes the 401 body for a rejected HTTP request in a role's native error
// format. authMiddleware calls it for both rejection paths (no authenticator configured, and a
// failed authentication), after setting the WWW-Authenticate header where appropriate, so a role can
// keep its error contract through the auth layer rather than emitting net/http's plain-text body. It
// must not log a request value; the middleware has already logged the method and outcome.
type unauthorizedResponder func(w http.ResponseWriter, r *http.Request)

// plainUnauthorized is the default 401 responder: net/http's plain-text "unauthorized" body. A role
// that has no structured error format (or does not need one on a 401) passes nil and gets this.
func plainUnauthorized(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}

// authMiddleware consults the Authenticator for every HTTP request and rejects an unauthenticated
// one with 401, attaching the Principal to the request context on success. The rejection body is
// written by onUnauthorized, so a role with a structured error contract (the FHIR role's
// OperationOutcome) keeps that contract through the auth layer; a nil onUnauthorized falls back to the
// plain-text body. The loopback reference daemons pass AllowAll(), so on a loopback bind every request
// is admitted as the anonymous principal; a non-loopback bind requires a real Authenticator (the bind
// policy enforces this at New). It logs the HTTP method and outcome only — never the URL query, which
// can carry PHI (PRD §9.1).
func authMiddleware(auth Authenticator, logger *zap.Logger, onUnauthorized unauthorizedResponder, next http.Handler) http.Handler {
	if onUnauthorized == nil {
		onUnauthorized = plainUnauthorized
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth == nil {
			// Defensive fail-closed: the bind policy refuses a non-loopback bind without an authenticator
			// and defaults a loopback bind to AllowAll, so auth is never nil in a correctly constructed
			// daemon. Should it ever be, reject rather than panic dereferencing it — an absent
			// authenticator must never read as an admitted request (PRD §9.1).
			logger.Info("http request rejected: no authenticator configured", zap.String("method", r.Method))
			onUnauthorized(w, r)
			return
		}
		principal, err := auth.AuthenticateHTTP(r.Context(), r)
		if err != nil {
			logger.Info("http request unauthenticated", zap.String("method", r.Method))
			w.Header().Set("WWW-Authenticate", "Bearer")
			onUnauthorized(w, r)
			return
		}
		ctx := context.WithValue(r.Context(), principalKey{}, principal)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// PrincipalFromContext returns the authenticated Principal attached to ctx by the HTTP auth
// middleware, and whether one was present. A handler downstream of authMiddleware uses it to read the
// caller's identity (its ID and scopes, never any secret material).
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	p, ok := ctx.Value(principalKey{}).(Principal)
	return p, ok
}
