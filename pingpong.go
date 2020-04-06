package pingpong

import (
	"fmt"
	"net/http"
)

// Router represents HTTP router.
type Router interface {
	HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request))
}

// Name is the service name.
const Name = "pingpong"

// New creates new pingpong service, attaches its handlers to r and returns the service.
func New(r Router) Server {
	svr := Server{router: r}
	svr.router.HandleFunc("/ping", svr.handlePing())
	return svr
}

// Server represents pingpong service.
type Server struct {
	router Router
}

func (svr Server) handlePing() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, "pong")
	}
}
