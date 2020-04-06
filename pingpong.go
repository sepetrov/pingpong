package pingpong

import (
	"fmt"
	"net/http"
)

type Server struct {

}

var _ http.Handler = Server{}

func (svr Server) ServeHTTP(w http.ResponseWriter, _ *http.Request) {
	fmt.Fprint(w, "pong")
}
