package main

import (
	"log"
	"net/http"
	"os"

	"gopkg.in/DataDog/dd-trace-go.v1/contrib/gorilla/mux"

	"github.com/sepetrov/pingpong"
)

func main() {
	router := mux.NewRouter(mux.WithServiceName(pingpong.Name))

	pingpong.New(routerAdapter{router})
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		log.Fatalf("HTTP_PORT not set")
	}

	log.Println("listening on port" + port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}

type routerAdapter struct {
	mux *mux.Router
}

func (r routerAdapter) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	r.mux.HandleFunc(pattern, handler)
}
