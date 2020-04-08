package main

import (
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
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
	for _, k := range []string{
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_REGION",
		"SQS_QUEUE",
	} {
		if os.Getenv(k) == "" {
			log.Fatalf("%s is required", k)
		}
	}

	port := os.Getenv("HTTP_PORT")
	if port == "" {
		port = defaultHTTPPort
	}

	svr, err := pingpong.New(
		sqs.New(session.Must(session.NewSession())),
		os.Getenv("SQS_QUEUE"),
		log.StandardLogger(),
	)
	if err != nil {
		log.Fatal(err)
	}

	router := mux.NewRouter(mux.WithServiceName("pingpong-server"))
	router.HandleFunc("/ping", svr.ServeHTTP)

	tracer.Start(tracer.WithAnalytics(true))
	defer tracer.Stop()

	log.Println("listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, router))
}
