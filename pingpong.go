package pingpong

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Router represents HTTP router.
type Router interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

// New creates new pingpong service, attaches its handlers to r and returns the service.
func New(r Router, l *log.Logger) Server {
	if l == nil {
		log.New(os.Stderr, "", 0)
	}

	svr := Server{router: r}

	logRequest := requestLogger{logger: svr.logger}.wrap

	svr.router.HandleFunc("/ping", logRequest(svr.handlePing()))

	return svr
}

// Server represents pingpong service.
type Server struct {
	router Router
	logger log.Logger
}

func (svr Server) handlePing() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		switch rand.Intn(10) {
		case 0:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		case 1:
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		case 2:
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		time.Sleep(time.Duration(rand.Intn(100)) * 30 * time.Millisecond)

		fmt.Fprint(w, "pong")
	}
}

type requestLogger struct {
	logger log.Logger
}

func (l requestLogger) wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%v\n", r.Header)
		next(w, r)
	}
}
