package service

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

type Message struct {
	MessageID string    `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

type SQSService struct {
	Client    *sqs.Client
	QueueName string
	QueueURL  string
	Log       *slog.Logger
}

func NewSQSService(ctx context.Context, client *sqs.Client, name, url string, log *slog.Logger) *SQSService {
	s := &SQSService{
		Client:    client,
		QueueName: name,
		QueueURL:  url,
		Log:       log,
	}
	if s.QueueURL == "" && s.QueueName != "" {
		qURL, err := s.resolveQueueURL(ctx)
		if err != nil {
			s.Log.Error("❌ Failed to resolve queue URL", "queue_name", s.QueueName, "error", err)
		} else {
			s.QueueURL = qURL
			s.Log.Info("✅ Resolved queue URL", "queue_name", s.QueueName, "queue_url", qURL)
		}
	}
	return s
}

func (s *SQSService) resolveQueueURL(ctx context.Context) (string, error) {
	out, err := s.Client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(s.QueueName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get queue URL for %s: %w", s.QueueName, err)
	}
	return *out.QueueUrl, nil
}

func (s *SQSService) Send(ctx context.Context, msg string) error {
	if s.QueueURL == "" {
		return fmt.Errorf("queue URL not set")
	}
	input := &sqs.SendMessageInput{
		QueueUrl:    &s.QueueURL,
		MessageBody: &msg,
	}
	if isFIFO(s.QueueName) {
		groupID := "default-group"
		input.MessageGroupId = &groupID
	}
	_, err := s.Client.SendMessage(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	s.Log.Info("📤 Message sent", "queue", s.QueueName)
	return nil
}

func (s *SQSService) Receive(ctx context.Context, max int32) ([]map[string]interface{}, error) {
	if s.QueueURL == "" {
		return nil, fmt.Errorf("queue URL not set")
	}

	// ⏱ Add a 10-second overall timeout for the receive operation
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var allMsgs []map[string]interface{}

	for {
		// Check if timeout or cancellation was triggered
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("receive operation timed out after 10s: %w", ctx.Err())
		default:
		}

		input := &sqs.ReceiveMessageInput{
			QueueUrl:            &s.QueueURL,
			MaxNumberOfMessages: max,
			VisibilityTimeout:   2, // short peek
			WaitTimeSeconds:     1, // quick poll
		}

		resp, err := s.Client.ReceiveMessage(ctx, input)
		if err != nil {

			// Distinguish between timeout vs API error
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				return nil, fmt.Errorf("receive operation timed out after 10s: %w", err)
			}
			return nil, fmt.Errorf("failed to receive messages: %w", err)
		}

		if len(resp.Messages) == 0 {
			break // no more messages in the queue
		}

		for _, m := range resp.Messages {
			msg := map[string]interface{}{
				"MessageId": *m.MessageId,
				"Body":      *m.Body,
			}
			allMsgs = append(allMsgs, msg)
		}
	}

	s.Log.Info("📥 Messages fetched", "count", len(allMsgs))
	return allMsgs, nil
}

func (s *SQSService) Purge(ctx context.Context) error {
	if s.QueueURL == "" {
		return fmt.Errorf("queue URL not set")
	}
	_, err := s.Client.PurgeQueue(ctx, &sqs.PurgeQueueInput{
		QueueUrl: &s.QueueURL,
	})
	if err != nil {
		return fmt.Errorf("failed to purge queue: %w", err)
	}
	s.Log.Info("✅ Queue purged successfully", "queue", s.QueueName)
	return nil
}

func (s *SQSService) Info(ctx context.Context) map[string]interface{} {
	info := map[string]interface{}{
		"queue_name": s.QueueName,
		"queue_url":  s.QueueURL,
		"status":     "ok",
	}
	out, err := s.Client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl: &s.QueueURL,
		AttributeNames: []types.QueueAttributeName{
			types.QueueAttributeNameApproximateNumberOfMessages,
			types.QueueAttributeNameApproximateNumberOfMessagesNotVisible,
			types.QueueAttributeNameApproximateNumberOfMessagesDelayed,
		},
	})
	if err != nil {
		s.Log.Error("failed to get queue attributes", "error", err)
		info["error"] = err.Error()
		info["status"] = "error"
		return info
	}
	parseInt := func(s string) int64 {
		n, _ := strconv.ParseInt(s, 10, 64)
		return n
	}
	visible := parseInt(out.Attributes[string(types.QueueAttributeNameApproximateNumberOfMessages)])
	notVisible := parseInt(out.Attributes[string(types.QueueAttributeNameApproximateNumberOfMessagesNotVisible)])
	delayed := parseInt(out.Attributes[string(types.QueueAttributeNameApproximateNumberOfMessagesDelayed)])
	info["number_of_messages"] = visible + notVisible + delayed
	info["approximate_number_of_messages"] = visible
	info["approximate_number_of_messages_not_visible"] = notVisible
	info["approximate_number_of_messages_delayed"] = delayed
	return info
}

func isFIFO(name string) bool {
	return strings.HasSuffix(name, ".fifo")
}
