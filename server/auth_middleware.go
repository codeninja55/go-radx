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

// authMiddleware consults the Authenticator for every HTTP request and rejects an unauthenticated
// one with 401 (a value-free body), attaching the Principal to the request context on success. The
// loopback reference daemons pass AllowAll(), so on a loopback bind every request is admitted as the
// anonymous principal; a non-loopback bind requires a real Authenticator (the bind policy enforces
// this at New). It logs the HTTP method and outcome only — never the URL query, which can carry PHI
// (PRD §9.1).
func authMiddleware(auth Authenticator, logger *zap.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth == nil {
			// Defensive fail-closed: the bind policy refuses a non-loopback bind without an authenticator
			// and defaults a loopback bind to AllowAll, so auth is never nil in a correctly constructed
			// daemon. Should it ever be, reject rather than panic dereferencing it — an absent
			// authenticator must never read as an admitted request (PRD §9.1).
			logger.Info("http request rejected: no authenticator configured", zap.String("method", r.Method))
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		principal, err := auth.AuthenticateHTTP(r.Context(), r)
		if err != nil {
			logger.Info("http request unauthenticated", zap.String("method", r.Method))
			w.Header().Set("WWW-Authenticate", "Bearer")
			http.Error(w, "unauthorized", http.StatusUnauthorized)
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
