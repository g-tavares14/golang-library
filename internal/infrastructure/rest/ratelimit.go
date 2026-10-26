package rest

import (
	"math"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter limita as requisições por IP com um balde de fichas para cada cliente (token bucket):
// cada requisição gasta uma ficha, e as fichas voltam num ritmo fixo até o tamanho do balde.
// É seguro para uso concorrente.
type RateLimiter struct {
	mutex     sync.Mutex          // trava o mapa e lastSweep: várias requisições mexem neles ao mesmo tempo
	visitors  map[string]*visitor // um balde por IP
	limit     rate.Limit          // ritmo de recuperação (por exemplo, 1 ficha a cada 12 s)
	burst     int                 // quantas tentativas seguidas cabem (por exemplo, 5)
	idleTTL   time.Duration       // depois de quanto tempo sem uso apagamos a entrada
	now       func() time.Time    // relógio; time.Now, a menos que um teste troque com WithClock
	lastSweep time.Time           // quando as entradas inativas foram removidas pela última vez
}

// visitor guarda o balde de um IP e quando ele foi visto pela última vez, para a limpeza das entradas inativas.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiterOption ajusta um RateLimiter ao criá-lo.
type RateLimiterOption func(*RateLimiter)

// WithClock troca o relógio do RateLimiter (o padrão é time.Now). Serve para os testes avançarem
// o tempo sem esperar de verdade.
func WithClock(now func() time.Time) RateLimiterOption {
	return func(limiter *RateLimiter) {
		limiter.now = now
	}
}

// NewRateLimiter cria um RateLimiter em que cada IP tem um balde de burst fichas, que recupera uma
// ficha a cada 1/limit segundo. Entradas de IPs sem requisições há mais de idleTTL são descartadas.
// burst precisa ser pelo menos 1.
func NewRateLimiter(limit rate.Limit, burst int, idleTTL time.Duration, options ...RateLimiterOption) *RateLimiter {
	limiter := &RateLimiter{
		visitors: make(map[string]*visitor),
		limit:    limit,
		burst:    burst,
		idleTTL:  idleTTL,
		now:      time.Now,
	}

	for _, option := range options {
		option(limiter)
	}
	limiter.lastSweep = limiter.now()

	return limiter
}

// Limit só deixa passar a requisição se o IP de origem ainda tiver fichas.
// Sem fichas, responde 429 com o cabeçalho Retry-After (em segundos) e não chama next.
func (limiter *RateLimiter) Limit(next http.HandlerFunc) http.HandlerFunc {
	return func(responseWriter http.ResponseWriter, request *http.Request) {
		allowed, retryAfter := limiter.allow(clientIP(request))
		if !allowed {
			seconds := max(1, int(math.Ceil(retryAfter.Seconds())))
			responseWriter.Header().Set("Retry-After", strconv.Itoa(seconds))
			writeError(responseWriter, http.StatusTooManyRequests, "too many requests, try again later")
			return
		}

		next(responseWriter, request)
	}
}

// allow gasta uma ficha do balde do IP. Se o balde está vazio, devolve false e quanto falta
// para a próxima ficha, sem gastar nada.
func (limiter *RateLimiter) allow(ip string) (bool, time.Duration) {
	limiter.mutex.Lock()
	defer limiter.mutex.Unlock()

	now := limiter.now()
	limiter.sweep(now)

	client, exists := limiter.visitors[ip]
	if !exists {
		client = &visitor{limiter: rate.NewLimiter(limiter.limit, limiter.burst)}
		limiter.visitors[ip] = client
	}
	client.lastSeen = now

	reservation := client.limiter.ReserveN(now, 1)
	if !reservation.OK() {
		// Só acontece com burst menor que 1, que NewRateLimiter documenta como inválido.
		return false, 0
	}

	if delay := reservation.DelayFrom(now); delay > 0 {
		// Sem ficha agora: devolve a reserva, porque a requisição não vai prosseguir.
		reservation.CancelAt(now)
		return false, delay
	}

	return true, 0
}

// sweep apaga as entradas sem uso há mais de idleTTL. Roda no máximo uma vez por idleTTL,
// dentro de allow, então não precisa de goroutine nem de encerramento. Quem chama segura o mutex.
func (limiter *RateLimiter) sweep(now time.Time) {
	if now.Sub(limiter.lastSweep) < limiter.idleTTL {
		return
	}

	for ip, client := range limiter.visitors {
		if now.Sub(client.lastSeen) > limiter.idleTTL {
			delete(limiter.visitors, ip)
		}
	}
	limiter.lastSweep = now
}

// clientIP devolve o IP da conexão, sem a porta. Os cabeçalhos X-Forwarded-For e X-Real-IP são
// ignorados de propósito: qualquer cliente pode forjá-los e ganhar um balde novo a cada requisição.
// Atrás de um proxy confiável, é o proxy que precisa informar o IP real de outro jeito.
func clientIP(request *http.Request) string {
	ip, _, err := net.SplitHostPort(request.RemoteAddr)
	if err != nil {
		// RemoteAddr sem porta: já é só o endereço.
		return request.RemoteAddr
	}

	return ip
}
