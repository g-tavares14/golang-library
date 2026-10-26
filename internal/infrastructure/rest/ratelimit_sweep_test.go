package rest

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// Este teste está no pacote rest (e não rest_test) porque confere o tamanho do mapa interno.
func TestRateLimiterSweepsIdleEntries(t *testing.T) {
	current := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	limiter := NewRateLimiter(rate.Every(time.Second), 3, time.Minute, WithClock(func() time.Time { return current }))

	for _, ip := range []string{"192.0.2.1", "192.0.2.2", "192.0.2.3"} {
		limiter.allow(ip)
	}
	if got := len(limiter.visitors); got != 3 {
		t.Fatalf("entradas = %d, want 3", got)
	}

	// Ainda dentro do idleTTL: ninguém é removido, e o 192.0.2.1 renova o seu último uso.
	current = current.Add(30 * time.Second)
	limiter.allow("192.0.2.1")
	if got := len(limiter.visitors); got != 3 {
		t.Errorf("antes do idleTTL: entradas = %d, want 3", got)
	}

	// 90 s depois do começo: 192.0.2.2 e .3 estão parados há 90 s (> idleTTL) e saem; o 192.0.2.1
	// ficou parado só 60 s (não passa do idleTTL) e fica, junto do IP novo.
	current = current.Add(60 * time.Second)
	limiter.allow("192.0.2.9")
	if got := len(limiter.visitors); got != 2 {
		t.Errorf("depois do idleTTL: entradas = %d, want 2 (192.0.2.1 e 192.0.2.9)", got)
	}
	for _, ip := range []string{"192.0.2.1", "192.0.2.9"} {
		if _, exists := limiter.visitors[ip]; !exists {
			t.Errorf("a entrada de %s deveria ter sobrado", ip)
		}
	}
}

func TestClientIP(t *testing.T) {
	tests := []struct {
		name       string
		remoteAddr string
		want       string
	}{
		{"IPv4 com porta", "192.0.2.1:54321", "192.0.2.1"},
		{"IPv6 com porta", "[2001:db8::1]:54321", "2001:db8::1"},
		{"sem porta", "192.0.2.1", "192.0.2.1"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			httpRequest := httptest.NewRequest(http.MethodGet, "/", nil)
			httpRequest.RemoteAddr = test.remoteAddr

			if got := clientIP(httpRequest); got != test.want {
				t.Errorf("clientIP() = %q, want %q", got, test.want)
			}
		})
	}
}
