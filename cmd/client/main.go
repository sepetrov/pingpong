package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

const defaultServerAddr = "http://localhost:8080"

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

	u, err := url.Parse(addr)
	if err != nil {
		log.Fatalln(err)
	}
	u.Path = "/ping"

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		<-ticker.C
		if err := ping(u); err != nil {
			log.Println(err)
		}
	}
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
