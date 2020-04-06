package main

import (
	"log"
	"net/http"
	"os"

	"github.com/sepetrov/pingpong"
)

func main() {
	svr := pingpong.Server{}
	port := os.Getenv("HTTP_PORT")
	if port == "" {
		log.Fatalf("HTTP_PORT not set")
	}

	log.Println("listening on port" + port)
	log.Fatal(http.ListenAndServe(":"+port, svr))
}
