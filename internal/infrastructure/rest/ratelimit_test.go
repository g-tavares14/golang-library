package rest_test

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"go-lang/internal/infrastructure/rest"
)

// fakeClock é um relógio que só anda quando o teste manda.
type fakeClock struct {
	mutex   sync.Mutex
	current time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{current: time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)}
}

func (clock *fakeClock) now() time.Time {
	clock.mutex.Lock()
	defer clock.mutex.Unlock()
	return clock.current
}

func (clock *fakeClock) advance(duration time.Duration) {
	clock.mutex.Lock()
	defer clock.mutex.Unlock()
	clock.current = clock.current.Add(duration)
}

// Um balde de 3 fichas que recupera 1 ficha a cada 12 segundos.
const (
	testBurst    = 3
	testInterval = 12 * time.Second
)

// newTestLimiter devolve o limitador e o handler protegido por ele; o handler responde 204.
func newTestLimiter(clock *fakeClock) (*rest.RateLimiter, http.HandlerFunc) {
	limiter := rest.NewRateLimiter(rate.Every(testInterval), testBurst, time.Minute, rest.WithClock(clock.now))
	handler := limiter.Limit(func(responseWriter http.ResponseWriter, _ *http.Request) {
		responseWriter.WriteHeader(http.StatusNoContent)
	})
	return limiter, handler
}

// request envia uma requisição como se viesse de remoteAddr (IP:porta).
func request(handler http.HandlerFunc, remoteAddr string, headers map[string]string) *httptest.ResponseRecorder {
	httpRequest := httptest.NewRequest(http.MethodPost, "/login", nil)
	httpRequest.RemoteAddr = remoteAddr
	for name, value := range headers {
		httpRequest.Header.Set(name, value)
	}

	recorder := httptest.NewRecorder()
	handler(recorder, httpRequest)
	return recorder
}

func TestRateLimiterAllowsUpToTheBurstThenRejects(t *testing.T) {
	_, handler := newTestLimiter(newFakeClock())

	for attempt := 1; attempt <= testBurst; attempt++ {
		if got := request(handler, "192.0.2.1:1000", nil).Code; got != http.StatusNoContent {
			t.Fatalf("tentativa %d: status = %d, want %d", attempt, got, http.StatusNoContent)
		}
	}

	recorder := request(handler, "192.0.2.1:1000", nil)
	if recorder.Code != http.StatusTooManyRequests {
		t.Fatalf("tentativa %d: status = %d, want %d", testBurst+1, recorder.Code, http.StatusTooManyRequests)
	}

	if got := recorder.Header().Get("Retry-After"); got != strconv.Itoa(int(testInterval.Seconds())) {
		t.Errorf("Retry-After = %q, want %q", got, strconv.Itoa(int(testInterval.Seconds())))
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", got)
	}
	if got := recorder.Body.String(); got == "" || got[0] != '{' {
		t.Errorf("o corpo do 429 deveria ser JSON, veio %q", got)
	}
}

func TestRateLimiterRejectedRequestDoesNotReachTheHandler(t *testing.T) {
	var calls int
	limiter := rest.NewRateLimiter(rate.Every(testInterval), 1, time.Minute, rest.WithClock(newFakeClock().now))
	handler := limiter.Limit(func(http.ResponseWriter, *http.Request) { calls++ })

	request(handler, "192.0.2.1:1000", nil)
	request(handler, "192.0.2.1:1000", nil)

	if calls != 1 {
		t.Errorf("o handler foi chamado %d vezes, want 1", calls)
	}
}

func TestRateLimiterKeepsEachIPSeparate(t *testing.T) {
	_, handler := newTestLimiter(newFakeClock())

	for range testBurst {
		request(handler, "192.0.2.1:1000", nil)
	}

	if got := request(handler, "192.0.2.1:1000", nil).Code; got != http.StatusTooManyRequests {
		t.Fatalf("primeiro IP esgotado: status = %d, want 429", got)
	}
	if got := request(handler, "192.0.2.2:1000", nil).Code; got != http.StatusNoContent {
		t.Errorf("outro IP: status = %d, want 204", got)
	}
}

func TestRateLimiterRefillsOverTime(t *testing.T) {
	clock := newFakeClock()
	_, handler := newTestLimiter(clock)

	for range testBurst {
		request(handler, "192.0.2.1:1000", nil)
	}
	if got := request(handler, "192.0.2.1:1000", nil).Code; got != http.StatusTooManyRequests {
		t.Fatalf("balde vazio: status = %d, want 429", got)
	}

	clock.advance(testInterval - time.Second)
	if got := request(handler, "192.0.2.1:1000", nil).Code; got != http.StatusTooManyRequests {
		t.Errorf("1 s antes da ficha voltar: status = %d, want 429", got)
	}

	clock.advance(time.Second)
	if got := request(handler, "192.0.2.1:1000", nil).Code; got != http.StatusNoContent {
		t.Errorf("quando a ficha volta: status = %d, want 204", got)
	}
	if got := request(handler, "192.0.2.1:1000", nil).Code; got != http.StatusTooManyRequests {
		t.Errorf("só voltou uma ficha: status = %d, want 429", got)
	}
}

// Tentar de novo enquanto está bloqueado não pode atrasar a recuperação nem gastar fichas.
func TestRateLimiterRejectedAttemptsDoNotDelayRecovery(t *testing.T) {
	clock := newFakeClock()
	_, handler := newTestLimiter(clock)

	for range testBurst {
		request(handler, "192.0.2.1:1000", nil)
	}
	for range 20 {
		request(handler, "192.0.2.1:1000", nil)
	}

	clock.advance(testInterval)
	if got := request(handler, "192.0.2.1:1000", nil).Code; got != http.StatusNoContent {
		t.Errorf("depois de insistir e esperar 1 intervalo: status = %d, want 204", got)
	}
}

func TestRateLimiterIgnoresSpoofableHeaders(t *testing.T) {
	_, handler := newTestLimiter(newFakeClock())

	for range testBurst {
		request(handler, "192.0.2.1:1000", nil)
	}

	forged := map[string]string{"X-Forwarded-For": "203.0.113.50", "X-Real-IP": "203.0.113.51"}
	if got := request(handler, "192.0.2.1:1000", forged).Code; got != http.StatusTooManyRequests {
		t.Errorf("cabeçalhos forjados: status = %d, want 429", got)
	}
}

func TestRateLimiterTreatsDifferentPortsAsTheSameIP(t *testing.T) {
	_, handler := newTestLimiter(newFakeClock())

	for port := range testBurst {
		request(handler, "192.0.2.1:"+strconv.Itoa(2000+port), nil)
	}

	if got := request(handler, "192.0.2.1:9999", nil).Code; got != http.StatusTooManyRequests {
		t.Errorf("mesmo IP, porta nova: status = %d, want 429", got)
	}
}

// Com o relógio parado não há recuperação: exatamente testBurst requisições concorrentes podem passar.
func TestRateLimiterIsSafeForConcurrentUse(t *testing.T) {
	_, handler := newTestLimiter(newFakeClock())

	var allowed atomic.Int32
	var group sync.WaitGroup
	for range 50 {
		group.Add(1)
		go func() {
			defer group.Done()
			if request(handler, "192.0.2.1:1000", nil).Code == http.StatusNoContent {
				allowed.Add(1)
			}
		}()
	}
	group.Wait()

	if got := allowed.Load(); got != testBurst {
		t.Errorf("requisições permitidas = %d, want %d", got, testBurst)
	}
}
