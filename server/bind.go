package server

import "net"

// loopbackHost is the IPv4 loopback the roles bind to by default. The protocol servers already
// default a bare ":port" to 127.0.0.1; the daemon binds this host explicitly so Addrs reports a
// concrete interface and the bind policy reasons about one host string (PRD §9.1).
const loopbackHost = "127.0.0.1"

// isLoopbackHost reports whether host names a loopback interface (127.0.0.0/8, ::1) or is the empty
// host that the protocol servers resolve to loopback. "localhost" resolves to loopback on every
// supported platform, so it is treated as loopback too. Any other host — a concrete interface IP or
// 0.0.0.0 — is non-loopback and triggers the explicit-Authenticator requirement.
func isLoopbackHost(host string) bool {
	if host == "" || host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

// joinHostPort composes the bind address a role's listener uses from the daemon's bind host and the
// role's port. A zero port yields ":0" (an OS-assigned port), which a test reads back through Addrs.
func joinHostPort(host string, port int) string {
	return net.JoinHostPort(host, itoa(port))
}

// itoa renders a non-negative port without importing strconv at every call site; a negative port is
// clamped to 0 so the listener takes an OS-assigned port rather than failing on a malformed address.
func itoa(port int) string {
	if port <= 0 {
		return "0"
	}
	var buf [10]byte
	i := len(buf)
	for port > 0 {
		i--
		buf[i] = byte('0' + port%10)
		port /= 10
	}
	return string(buf[i:])
}
