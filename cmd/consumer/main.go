package main

import (
	"encoding/json"
	"math/rand"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	log "github.com/sirupsen/logrus"
	awstrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/aws/aws-sdk-go/aws"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

const serviceName = "pingpong-consumer"

func init() {
	log.SetFormatter(&log.JSONFormatter{})
	rand.Seed(time.Now().UnixNano())
}

func main() {
	for _, k := range []string{
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
		"AWS_REGION",
		"SQS_QUEUE_URL",
	} {
		if os.Getenv(k) == "" {
			log.Fatalf("%s is required", k)
		}
	}

	tracer.Start(tracer.WithAnalytics(true), tracer.WithServiceName(serviceName))
	defer tracer.Stop()

	sess := awstrace.WrapSession(
		session.Must(session.NewSession()),
		awstrace.WithServiceName(serviceName),
	)

	queue := os.Getenv("SQS_QUEUE_URL")
	svc := sqs.New(sess)
	for {
		work(svc, queue)
	}
}

func work(svc *sqs.SQS, queue string) {
	span := tracer.StartSpan("process_message")
	defer span.Finish()

	// Receive message.
	res, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
		MessageAttributeNames: []*string{
			aws.String("dd.trace_id"),
			aws.String("dd.span_id"),
			aws.String("span_ctx"),
		},
		QueueUrl:            aws.String(queue),
		MaxNumberOfMessages: aws.Int64(1),
		VisibilityTimeout:   aws.Int64(10),
		WaitTimeSeconds:     aws.Int64(0),
	})

	fields := log.Fields{"sqs_queue": queue}

	if err != nil {
		log.WithFields(fields).Error("sqs: read message: ", err)
		return
	}

	if len(res.Messages) == 0 {
		log.WithFields(fields).Info("no messages")
		time.Sleep(10 * time.Second)
		return
	}

	span.SetBaggageItem("dd.trace_id", *res.Messages[0].MessageAttributes["dd.trace_id"].StringValue)
	span.SetBaggageItem("dd.span_id", *res.Messages[0].MessageAttributes["dd.span_id"].StringValue)

	fields["dd.trace_id"] = *res.Messages[0].MessageAttributes["dd.trace_id"].StringValue
	fields["dd.span_id"] = *res.Messages[0].MessageAttributes["dd.span_id"].StringValue
	fields["sqs_message_id"] = *res.Messages[0].MessageId

	// Start child span of the message trace.
	var carr map[string]string
	if err := json.Unmarshal([]byte(*res.Messages[0].MessageAttributes["span_ctx"].StringValue), &carr); err != nil {
		log.WithFields(fields).Warnf("cannot decode span context: %v", err)
	} else {
		sctx, err := tracer.Extract(tracer.TextMapCarrier(carr))
		if err != nil {
			log.WithFields(fields).Warnf("cannot extract span context: %v", err)
		} else {

			span = tracer.StartSpan("process_message", tracer.ChildOf(sctx))
			defer span.Finish()
		}
	}

	// Do something time-consuming with the message.
	time.Sleep(time.Duration(rand.Intn(5)) * 100 * time.Millisecond)

	// Delete the message.
	_, err = svc.DeleteMessage(&sqs.DeleteMessageInput{
		QueueUrl:      aws.String(queue),
		ReceiptHandle: res.Messages[0].ReceiptHandle,
	})
	if err != nil {
		log.WithFields(fields).Error("sqs: delete message: %v", err)
		return
	}

	log.WithFields(fields).Info("message processed")
}
