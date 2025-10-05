package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// Message represents a simplified SQS message form (kept for potential future use).
type Message struct {
	MessageID string    `json:"id"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at,omitempty"`
}

// SQSService wraps SQS operations with configuration and logging.
type SQSService struct {
	Client                   *sqs.Client
	QueueName                string
	QueueURL                 string
	Log                      *slog.Logger
}

func NewSQSService(ctx context.Context, client *sqs.Client, queueName, queueURL string, log *slog.Logger) *SQSService {
	log.Debug("creating SQS service", "queue_name", queueName, "queue_url", queueURL)

	s := &SQSService{
		Client:                   client,
		QueueName:                queueName,
		QueueURL:                 queueURL,
		Log:                      log,
	}

	// If AWS credentials or region are missing, skip SQS init
	if client == nil {
		s.Log.Warn("AWS client not provided. Running without SQS connection.")
		return s
	}

	// Try to resolve the queue URL (only if name given)
	if queueURL == "" {

		resp, err := client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{QueueName: &queueName})
		if err != nil {
			s.Log.Warn("unable to resolve SQS queue URL", "queue_name", queueName, "error", err)
			return s
		}

		queueURL = *resp.QueueUrl
		s.Log.Info("resolved queue URL", "queue_name", queueName, "queue_url", s.QueueURL)

	}

	return s
}

// Send publishes a message to the queue (adds group id if FIFO).
func (s *SQSService) Send(ctx context.Context, msg string) error {
	s.Log.Debug("sending message", "msg", msg)

	if s.QueueURL == "" {
		return fmt.Errorf("queue URL not set")
	}
	if msg == "" {
		return fmt.Errorf("empty message body")
	}

	// Set a timeout for each operation
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	input := &sqs.SendMessageInput{
		QueueUrl:    &s.QueueURL,
		MessageBody: &msg,
	}

	// Using a default for now
	if isFIFO(s.QueueURL) {
		groupID := "default-group"
		input.MessageGroupId = &groupID
	}

	if _, err := s.Client.SendMessage(ctx, input); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	s.Log.Info("message sent", "queue", s.QueueName)

	return nil
}

// ReceiveAll behaves in loop mode regardless,
// aggregating batches until an empty batch, iteration cap, or timeout occurs.
func (s *SQSService) ReceiveAll(ctx context.Context, max int32) ([]map[string]interface{}, error) {
	s.Log.Debug("receiving messages", "max", max)

	if s.QueueURL == "" {
		return nil, fmt.Errorf("queue URL not set")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	start := time.Now()
	var allMsgs []map[string]interface{}

	doReceive := func(rc context.Context) (int, error) {
		input := &sqs.ReceiveMessageInput{
			QueueUrl:            &s.QueueURL,
			VisibilityTimeout:   10,
			WaitTimeSeconds:     5,
		}

		resp, err := s.Client.ReceiveMessage(rc, input)
		if err != nil {
			return 0, err
		}

		for _, m := range resp.Messages {
			allMsgs = append(allMsgs, map[string]interface{}{
				"MessageId": *m.MessageId,
				"Body":      *m.Body,
			})
		}

		return len(resp.Messages), nil
	}

	const maxIterations = 25
	for iteration := 1; iteration <= maxIterations; iteration++ {
		select {
		case <-ctx.Done():
			if len(allMsgs) > 0 {
				s.Log.Warn("receiveAll cancelled after partial retrieval", "count", len(allMsgs))
				goto END
			}
			return nil, fmt.Errorf("receive operation timed out: %w", ctx.Err())
		default:
		}

		n, err := doReceive(ctx)
		if err != nil {
			if (errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)) && len(allMsgs) > 0 {
				s.Log.Warn("receiveAll timeout after partial retrieval", "count", len(allMsgs))
				break
			}
			if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
				return nil, fmt.Errorf("receive operation timed out: %w", err)
			}
			return nil, fmt.Errorf("failed to receive messages: %w", err)
		}

		if n == 0 {
			break
		}

		if s.Log.Enabled(ctx, slog.LevelDebug) {
			s.Log.Debug("receiveAll batch", "batch_count", n, "total", len(allMsgs), "iteration", iteration)
		}

		if iteration == maxIterations {
			s.Log.Warn("receiveAll iteration cap reached", "cap", maxIterations, "count", len(allMsgs))
		}
	}
END:
	elapsed := time.Since(start)
	s.Log.Info("messages fetched",
		"count", len(allMsgs),
		"elapsed_ms", elapsed.Milliseconds(),
	)
	return allMsgs, nil
}

// Purge deletes all messages currently in the queue.
func (s *SQSService) Purge(ctx context.Context) error {
	s.Log.Debug("purging queue", "queue", s.QueueName)

	if s.QueueURL == "" {
		return fmt.Errorf("queue URL not set")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if _, err := s.Client.PurgeQueue(ctx, &sqs.PurgeQueueInput{QueueUrl: &s.QueueURL}); err != nil {
		return fmt.Errorf("failed to purge queue: %w", err)
	}

	s.Log.Info("queue purged", "queue", s.QueueName)
	return nil
}

// Info returns summary attributes for the queue (approximate counts).
func (s *SQSService) Info(ctx context.Context) map[string]interface{} {
	s.Log.Debug("fetching queue info", "queue_name", s.QueueName)

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	info := map[string]interface{}{
		"queue_name": s.QueueName,
		"queue_url":  s.QueueURL,
		"status":     "ok",
	}

	if s.Client == nil || s.QueueURL == "" {
		info["status"] = "not_connected"
		return info
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

	parseInt := func(v string) int64 {
		n, _ := strconv.ParseInt(v, 10, 64)
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

// isFIFO returns true if queue name contains fifo
func isFIFO(name string) bool {
	slog.Debug("checking if FIFO", "queue_name", name)
	return strings.Contains(name, "fifo")
}
