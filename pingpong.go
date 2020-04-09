package pingpong

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/sqs"
	log "github.com/sirupsen/logrus"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// New creates new pingpong service, attaches its handlers to r and returns the service.
func New(s *sqs.SQS, queue string, l *log.Logger) (Server, error) {
	if s == nil {
		return Server{}, errors.New("SQS client is required")
	}
	if queue == "" {
		return Server{}, errors.New("SQS queue is required")
	}
	if l == nil {
		l = log.New()
		l.SetFormatter(&log.JSONFormatter{})
	}

	svr := Server{
		sqs:    s,
		queue:  queue,
		logger: l,
	}

	return svr, nil
}

// Server represents pingpong service.
type Server struct {
	sqs    *sqs.SQS
	queue  string
	logger *log.Logger
}

func (svr Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	logRequest := requestLogger{logger: svr.logger}.wrap
	handler := logRequest(svr.handlePing())
	handler.ServeHTTP(w, r)
}

func (svr Server) handlePing() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch rand.Intn(10) {
		case 0:
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		case 1:
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		case 2:
			http.Error(w, http.StatusText(http.StatusTooManyRequests), http.StatusTooManyRequests)
			return
		}

		time.Sleep(time.Duration(rand.Intn(100)) * 30 * time.Millisecond)

		span, _ := tracer.StartSpanFromContext(r.Context(), "send_message", tracer.Tag("queue", svr.queue))
		defer span.Finish()

		fields := log.Fields{
			"dd.trace_id": fmt.Sprintf("%d", span.Context().TraceID()),
			"dd.span_id":  fmt.Sprintf("%d", span.Context().SpanID()),
		}

		carr := tracer.TextMapCarrier(map[string]string{})
		err := tracer.Inject(span.Context(), carr)
		jcarr, err := json.Marshal(carr)
		if err != nil {
			log.WithFields(fields).Warnf("cannot encode span context: %v", err)
			jcarr = []byte(`{}`)
		}

		out, err := svr.sqs.SendMessage(&sqs.SendMessageInput{
			MessageAttributes: map[string]*sqs.MessageAttributeValue{
				"dd.trace_id": {
					DataType:    aws.String("Number"),
					StringValue: aws.String(fmt.Sprintf("%d", span.Context().TraceID())),
				},
				"dd.span_id": {
					DataType:    aws.String("Number"),
					StringValue: aws.String(fmt.Sprintf("%d", span.Context().SpanID())),
				},
				"span_ctx": {
					DataType:    aws.String("String"),
					StringValue: aws.String(string(jcarr)),
				},
			},
			MessageBody: aws.String("ping"),
			QueueUrl:    aws.String(svr.queue),
		})

		if err != nil {
			log.WithFields(fields).Errorf("sqs: send message: %v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		fields["sqs_message_id"] = *out.MessageId
		log.WithFields(fields).Info("message sent")

		fmt.Fprint(w, "pong")
	}
}

type requestLogger struct {
	logger *log.Logger
}

func (l requestLogger) wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rw := responseWrapper{
			ResponseWriter: w,
			statusCode:     http.StatusOK,
		}

		next(&rw, r)

		fields := log.Fields{
			"status_code":     rw.statusCode,
			"status_text":     http.StatusText(rw.statusCode),
			"request_headers": r.Header,
		}
		if span, ok := tracer.SpanFromContext(r.Context()); ok {
			fields["dd.trace_id"] = fmt.Sprintf("%d", span.Context().TraceID())
			fields["dd.span_id"] = fmt.Sprintf("%d", span.Context().SpanID())
		}

		if rw.statusCode >= 400 {
			l.logger.WithFields(fields).Error("something went wrong")
		} else {
			l.logger.WithFields(fields).Info("all good")
		}
	}
}

type responseWrapper struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWrapper) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}
