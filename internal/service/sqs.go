package service

import (
	"context"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type SQSService struct {
	Client   *sqs.Client
	QueueURL string
	Log      *slog.Logger
}

func NewSQSService(ctx context.Context, client *sqs.Client, queueName, queueURL string, log *slog.Logger) *SQSService {
	if queueURL == "" {
		resp, err := client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{QueueName: &queueName})
		if err != nil {
			log.Error("failed to resolve queue URL", "queueName", queueName, "err", err)
			panic(err)
		}
		queueURL = *resp.QueueUrl
	}
	log.Info("resolved queue URL", "queueURL", queueURL)
	return &SQSService{Client: client, QueueURL: queueURL, Log: log}
}

func (s *SQSService) Send(ctx context.Context, msg string) error {
	s.Log.Info("sending message", "msg", msg)
	_, err := s.Client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    &s.QueueURL,
		MessageBody: &msg,
	})
	return err
}

func (s *SQSService) Receive(ctx context.Context, max int32) ([]string, error) {
	resp, err := s.Client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            &s.QueueURL,
		MaxNumberOfMessages: max,
		WaitTimeSeconds:     1,
	})
	if err != nil {
		return nil, err
	}
	var msgs []string
	for _, m := range resp.Messages {
		msgs = append(msgs, *m.Body)
	}
	s.Log.Info("received messages", "count", len(msgs))
	return msgs, nil
}
