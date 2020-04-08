package main

import (
	"context"
	"errors"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

const defaultServerAddr = "http://localhost:8080"

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	tracer.Start(
		tracer.WithServiceName("pingpong-client"),
		tracer.WithAnalytics(true),
		tracer.WithDebugMode(true),
	)
	defer tracer.Stop()

	addr := os.Getenv("SERVER_ADDR")
	if addr == "" {
		addr = defaultServerAddr
	}

	addrs, err := urls(addr)
	if err != nil {
		log.Fatalln(err)
	}

	for _, u := range addrs {
		go func(u *url.URL) {
			for {
				time.Sleep(time.Duration(rand.Intn(10)) * time.Millisecond)
				if err := ping(u); err != nil {
					log.Println(err)
				}
			}
		}(u)
	}

	ch := make(chan struct{})
	<-ch
}

func ping(u *url.URL) error {
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	span := tracer.StartSpan("pinging")
	defer span.Finish()

	ctx := tracer.ContextWithSpan(context.Background(), span)
	req = req.WithContext(ctx)

	// Inject the span Context in the Request headers
	err = tracer.Inject(span.Context(), tracer.HTTPHeadersCarrier(req.Header))
	if err != nil {
		return err
	}
	_, err = http.DefaultClient.Do(req)

	return err
}

func urls(s string) ([]*url.URL, error) {
	if s == "" {
		return nil, errors.New("urls: s is required")
	}

	var uu []*url.URL

	for _, addr := range strings.Split(s, ",") {
		u, err := url.Parse(addr)
		if err != nil {
			return nil, err
		}
		u.Path = "/ping"

		uu = append(uu, u)
	}

	return uu, nil
}
