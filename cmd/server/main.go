package main

import (
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"

	"gopkg.in/DataDog/dd-trace-go.v1/contrib/gorilla/mux"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/sepetrov/pingpong"
)

const defaultHTTPPort = "8080"

func init() {
	log.SetFormatter(&log.JSONFormatter{})
}

func main() {
	tracer.Start(
		tracer.WithAnalytics(true),
	)
	defer tracer.Stop()

	router := mux.NewRouter(mux.WithServiceName("pingpong-server"))

	pingpong.New(routerAdapter{router}, log.StandardLogger())
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = defaultHTTPPort
	}

	log.Println("listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

type routerAdapter struct {
	mux *mux.Router
}

func (r routerAdapter) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	r.mux.HandleFunc(pattern, handler)
}
