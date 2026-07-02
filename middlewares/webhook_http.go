package middlewares

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
)

// maxWebhookBodyBytes caps the request body. Telegram updates are small; this
// protects against oversized/garbage payloads.
const maxWebhookBodyBytes = 1 << 20 // 1 MiB

// telegramCIDRs are the official IP ranges Telegram sends webhook requests from.
// See https://core.telegram.org/bots/webhooks (Webhook IPs).
var telegramCIDRs = []string{
	"149.154.160.0/20",
	"91.108.4.0/22",
}

// telegramNets is the parsed form of telegramCIDRs, built once at init.
var telegramNets = parseCIDRs(telegramCIDRs)

func parseCIDRs(cidrs []string) []*net.IPNet {
	nets := make([]*net.IPNet, 0, len(cidrs))
	for _, c := range cidrs {
		_, n, err := net.ParseCIDR(c)
		if err != nil {
			// Static, known-good constants — a parse error is a programming bug.
			panic("invalid Telegram CIDR " + c + ": " + err.Error())
		}
		nets = append(nets, n)
	}
	return nets
}

// PostOnly rejects any method other than POST with 405, and caps the body size.
func PostOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		r.Body = http.MaxBytesReader(w, r.Body, maxWebhookBodyBytes)
		next.ServeHTTP(w, r)
	})
}

// TelegramIPAllowlist rejects requests whose source IP is not within Telegram's
// published ranges. When trustForwardedFor is true, the left-most address in
// X-Forwarded-For is used instead of RemoteAddr (only enable behind a trusted
// reverse proxy that sets this header).
func TelegramIPAllowlist(trustForwardedFor bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := clientIP(r, trustForwardedFor)
			if ip == nil || !ipAllowed(ip) {
				slog.Warn("webhook request from disallowed IP",
					"remote_addr", r.RemoteAddr,
					"forwarded_for", r.Header.Get("X-Forwarded-For"),
				)
				http.Error(w, "forbidden", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func ipAllowed(ip net.IP) bool {
	for _, n := range telegramNets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// clientIP extracts the source IP from the request.
func clientIP(r *http.Request, trustForwardedFor bool) net.IP {
	if trustForwardedFor {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			// Left-most entry is the original client.
			first := xff
			if i := strings.IndexByte(xff, ','); i >= 0 {
				first = xff[:i]
			}
			if ip := net.ParseIP(strings.TrimSpace(first)); ip != nil {
				return ip
			}
		}
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// RemoteAddr may already be a bare IP.
		host = r.RemoteAddr
	}
	return net.ParseIP(host)
}
