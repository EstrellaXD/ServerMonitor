package api

import (
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"slices"
	"strings"
)

// hostAllowed reports whether host (without port) may address this server:
// an IP literal, localhost, a *.local mDNS name, or an entry of allowed.
// A DNS rebinding attack uses the attacker's own name, which fails here.
func hostAllowed(host string, allowed []string) bool {
	host = strings.ToLower(strings.TrimSuffix(strings.TrimPrefix(host, "["), "]"))
	if _, err := netip.ParseAddr(host); err == nil {
		return true
	}
	return host == "localhost" || strings.HasSuffix(host, ".local") || slices.Contains(allowed, host)
}

// hostAllowlist rejects requests whose Host header fails hostAllowed. The
// list is read on each request, so /api/reload applies a changed list.
func hostAllowlist(state *AppState) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, _, err := net.SplitHostPort(r.Host)
			if err != nil {
				host = r.Host // no port
			}
			if !hostAllowed(host, state.Manager.AllowedHosts()) {
				errorJSON(w, http.StatusForbidden, "host not allowed")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// originAllowed is the /ws origin check. Native clients send no Origin; a
// browser page must come from a host that hostAllowed accepts.
func originAllowed(origin string, allowed []string) bool {
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	if err != nil || u.Hostname() == "" {
		return false
	}
	return hostAllowed(u.Hostname(), allowed)
}
