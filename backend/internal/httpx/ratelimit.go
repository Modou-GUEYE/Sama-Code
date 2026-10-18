package httpx

import (
	"net"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// RateLimiter limite le débit par adresse IP, en mémoire. Suffisant pour une instance ;
// à remplacer par un stockage partagé (Redis…) si l'API est répliquée.
type RateLimiter struct {
	mu        sync.Mutex
	visitors  map[string]*visitor
	limit     rate.Limit
	burst     int
	lastSweep time.Time
}

type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

const visitorTTL = 10 * time.Minute

// NewRateLimiter autorise `burst` requêtes immédiates puis une requête toutes les `every`.
func NewRateLimiter(every time.Duration, burst int) *RateLimiter {
	return &RateLimiter{visitors: map[string]*visitor{}, limit: rate.Every(every), burst: burst}
}

func (l *RateLimiter) allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if now.Sub(l.lastSweep) > time.Minute {
		for k, v := range l.visitors {
			if now.Sub(v.lastSeen) > visitorTTL {
				delete(l.visitors, k)
			}
		}
		l.lastSweep = now
	}

	v, ok := l.visitors[key]
	if !ok {
		v = &visitor{limiter: rate.NewLimiter(l.limit, l.burst)}
		l.visitors[key] = v
	}
	v.lastSeen = now
	return v.limiter.Allow()
}

func (l *RateLimiter) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !l.allow(clientIP(r)) {
			WriteError(w, r, ErrTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// clientIP lit RemoteAddr, déjà réécrit par middleware.RealIP si le proxy est de confiance.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
