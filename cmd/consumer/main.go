package main

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
	log "github.com/sirupsen/logrus"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func init() {
	log.SetFormatter(&log.JSONFormatter{})
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

	tracer.Start(tracer.WithAnalytics(true), tracer.WithServiceName("pingpong-consumer"))
	defer tracer.Stop()

	queue := os.Getenv("SQS_QUEUE_URL")
	svc := sqs.New(session.Must(session.NewSession()))
	for {
		work(svc, queue)
	}
}

func work(svc *sqs.SQS, queue string) {
	span := tracer.StartSpan("process_message")
	defer span.Finish()

	res, err := svc.ReceiveMessage(&sqs.ReceiveMessageInput{
		AttributeNames: []*string{
			aws.String(sqs.MessageSystemAttributeNameSentTimestamp),
		},
		MessageAttributeNames: []*string{
			aws.String(sqs.QueueAttributeNameAll),
			aws.String("dd.trace_id"),
			aws.String("dd.span_id"),
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

	fields["dd.trace_id"] = *res.Messages[0].MessageAttributes["dd.trace_id"].StringValue
	fields["dd.span_id"] = *res.Messages[0].MessageAttributes["dd.span_id"].StringValue
	fields["sqs_message_id"] = *res.Messages[0].MessageId

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
