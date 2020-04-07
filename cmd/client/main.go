package main

import (
	"log"
	"net/http"
	"net/url"
	"os"
	"time"
)

const defaultServerAddr = "http://localhost:8080"

func main() {
	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		addr = defaultServerAddr
	}

	u, err := url.Parse(addr)
	if err != nil {
		log.Fatalln(err)
	}
	u.Path = "/ping"

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		if _, err := http.DefaultClient.Get(u.String()); err != nil {
			log.Println(err)
		}
	}
}
