package main

import (
	"log"
	"net/http"
	"os"

	"gopkg.in/DataDog/dd-trace-go.v1/contrib/gorilla/mux"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/sepetrov/pingpong"
)

const defaultHTTPPort = "8080"

func main() {
	tracer.Start(
		tracer.WithAnalytics(true),
		tracer.WithDebugMode(true),
	)
	defer tracer.Stop()

	router := mux.NewRouter(mux.WithServiceName("pingpong-server"))

	pingpong.New(routerAdapter{router}, nil)
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
