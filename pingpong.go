package pingpong

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
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
		l = log.New()
		l.SetFormatter(&log.JSONFormatter{})
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
		rw := responseWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}
		next(&rw, r)

		fields := log.Fields{
			"status_code": rw.statusCode,
			"status_text": http.StatusText(rw.statusCode),
		}
		if span, ok := tracer.SpanFromContext(r.Context()); ok {
			fields["dd.trace_id"] = span.Context().TraceID()
			fields["dd.span_id"] = span.Context().SpanID()
		}

		if rw.statusCode >= 400 {
			l.logger.WithFields(fields).Error("something went wrong")
		} else {
			l.logger.WithFields(fields).Info("all good")
		}
	}
}

type responseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
