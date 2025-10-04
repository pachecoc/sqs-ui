package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
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
	DefaultGroupID           string
	OperationTimeout         time.Duration
	MaxReceiveBatch          int32
	ReceiveWaitSeconds       int32
	ReceiveVisibilityTimeout int32
	SingleCallReceive        bool
}

// NewSQSService constructs a service instance and resolves queue URL if only name provided.
func NewSQSService(
	ctx context.Context,
	client *sqs.Client,
	name, url string,
	log *slog.Logger,
	groupID string,
	opTimeout time.Duration,
	maxReceive int32,
	waitSeconds int32,
	visibilitySeconds int32,
	singleCall bool,
) *SQSService {
	s := &SQSService{
		Client:                   client,
		QueueName:                name,
		QueueURL:                 url,
		Log:                      log,
		DefaultGroupID:           groupID,
		OperationTimeout:         opTimeout,
		MaxReceiveBatch:          maxReceive,
		ReceiveWaitSeconds:       waitSeconds,
		ReceiveVisibilityTimeout: visibilitySeconds,
		SingleCallReceive:        singleCall,
	}
	if s.QueueURL == "" && s.QueueName != "" {
		qURL, err := s.resolveQueueURL(ctx)
		if err != nil {
			s.Log.Error("failed to resolve queue URL", "queue_name", s.QueueName, "error", err)
		} else {
			s.QueueURL = qURL
			s.Log.Info("resolved queue URL", "queue_name", s.QueueName, "queue_url", qURL)
		}
	}
	return s
}

// resolveQueueURL fetches the queue URL from AWS given a queue name.
func (s *SQSService) resolveQueueURL(ctx context.Context) (string, error) {
	out, err := s.Client.GetQueueUrl(ctx, &sqs.GetQueueUrlInput{
		QueueName: aws.String(s.QueueName),
	})
	if err != nil {
		return "", fmt.Errorf("failed to get queue URL for %s: %w", s.QueueName, err)
	}
	return *out.QueueUrl, nil
}

// Send publishes a message to the queue (adds group id if FIFO).
func (s *SQSService) Send(ctx context.Context, msg string) error {
	if s.QueueURL == "" {
		return fmt.Errorf("queue URL not set")
	}
	if msg == "" {
		return fmt.Errorf("empty message body")
	}

	ctx, cancel := context.WithTimeout(ctx, s.OperationTimeout)
	defer cancel()

	input := &sqs.SendMessageInput{
		QueueUrl:    &s.QueueURL,
		MessageBody: &msg,
	}
	if isFIFO(s.QueueName) {
		groupID := s.DefaultGroupID
		input.MessageGroupId = &groupID
	}

	if _, err := s.Client.SendMessage(ctx, input); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}
	s.Log.Info("message sent", "queue", s.QueueName)
	return nil
}

// Receive retrieves up to max messages (capped by configuration).
// Single-call mode performs one ReceiveMessage call; loop mode keeps calling until:
// - an empty batch is returned
// - iteration cap reached
// - context timeout/cancel
// If timeout/cancel occurs after at least one batch, partial messages are returned without error.
func (s *SQSService) Receive(ctx context.Context, max int32) ([]map[string]interface{}, error) {
	if s.QueueURL == "" {
		return nil, fmt.Errorf("queue URL not set")
	}
	if max <= 0 || max > s.MaxReceiveBatch {
		max = s.MaxReceiveBatch
	}

	timeout := s.OperationTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	var allMsgs []map[string]interface{}

	// doReceive performs a single SQS ReceiveMessage call and appends to allMsgs.
	doReceive := func(rc context.Context) (int, error) {
		input := &sqs.ReceiveMessageInput{
			QueueUrl:            &s.QueueURL,
			MaxNumberOfMessages: max,
			VisibilityTimeout:   s.ReceiveVisibilityTimeout,
			WaitTimeSeconds:     s.ReceiveWaitSeconds,
		}
		resp, err := s.Client.ReceiveMessage(rc, input)
		if err != nil {
			return 0, err
		}
		for _, m := range resp.Messages {
			// Only safe dereferences (SQS always sets IDs and Body)
			allMsgs = append(allMsgs, map[string]interface{}{
				"MessageId": *m.MessageId,
				"Body":      *m.Body,
			})
		}
		return len(resp.Messages), nil
	}

	if s.SingleCallReceive {
		if s.Log.Enabled(ctx, slog.LevelDebug) {
			s.Log.Debug("receive single-call", "max", max)
		}
		_, err := doReceive(ctx)
		if err != nil {
			switch {
			case (errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)) && len(allMsgs) > 0:
				s.Log.Warn("receive timeout after partial retrieval", "count", len(allMsgs))
			case errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled):
				return nil, fmt.Errorf("receive operation timed out: %w", err)
			default:
				return nil, fmt.Errorf("failed to receive messages: %w", err)
			}
		}
	} else {
		if s.Log.Enabled(ctx, slog.LevelDebug) {
			s.Log.Debug("receive loop", "max", max)
		}
		const maxIterations = 25
		for iteration := 1; iteration <= maxIterations; iteration++ {
			// Fast cancellation check
			select {
			case <-ctx.Done():
				if len(allMsgs) > 0 {
					s.Log.Warn("receive cancelled after partial retrieval", "count", len(allMsgs))
					break
				}
				return nil, fmt.Errorf("receive operation timed out: %w", ctx.Err())
			default:
			}

			n, err := doReceive(ctx)
			if err != nil {
				switch {
				case (errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled)) && len(allMsgs) > 0:
					s.Log.Warn("receive timeout after partial retrieval", "count", len(allMsgs))
					break
				case errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled):
					return nil, fmt.Errorf("receive operation timed out: %w", err)
				default:
					return nil, fmt.Errorf("failed to receive messages: %w", err)
				}
			}

			if n == 0 {
				// Empty batch means no more immediately available
				break
			}

			if s.Log.Enabled(ctx, slog.LevelDebug) {
				s.Log.Debug("receive batch", "batch_count", n, "total", len(allMsgs), "iteration", iteration)
			}

			if iteration == maxIterations {
				s.Log.Warn("receive loop iteration cap reached", "cap", maxIterations, "count", len(allMsgs))
			}
		}
	}

	elapsed := time.Since(start)
	s.Log.Info("messages fetched",
		"count", len(allMsgs),
		"elapsed_ms", elapsed.Milliseconds(),
		"single_call", s.SingleCallReceive,
	)
	return allMsgs, nil
}

// ReceiveAll behaves like Receive in loop mode regardless of SingleCallReceive flag,
// aggregating batches until an empty batch, iteration cap, or timeout occurs.
func (s *SQSService) ReceiveAll(ctx context.Context, max int32) ([]map[string]interface{}, error) {
	if s.QueueURL == "" {
		return nil, fmt.Errorf("queue URL not set")
	}
	if max <= 0 || max > s.MaxReceiveBatch {
		max = s.MaxReceiveBatch
	}

	timeout := s.OperationTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	start := time.Now()
	var allMsgs []map[string]interface{}

	doReceive := func(rc context.Context) (int, error) {
		input := &sqs.ReceiveMessageInput{
			QueueUrl:            &s.QueueURL,
			MaxNumberOfMessages: max,
			VisibilityTimeout:   s.ReceiveVisibilityTimeout,
			WaitTimeSeconds:     s.ReceiveWaitSeconds,
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
	s.Log.Info("messages fetched (receiveAll)",
		"count", len(allMsgs),
		"elapsed_ms", elapsed.Milliseconds(),
	)
	return allMsgs, nil
}

// Purge deletes all messages currently in the queue.
func (s *SQSService) Purge(ctx context.Context) error {
	if s.QueueURL == "" {
		return fmt.Errorf("queue URL not set")
	}
	ctx, cancel := context.WithTimeout(ctx, s.OperationTimeout)
	defer cancel()

	if _, err := s.Client.PurgeQueue(ctx, &sqs.PurgeQueueInput{QueueUrl: &s.QueueURL}); err != nil {
		return fmt.Errorf("failed to purge queue: %w", err)
	}
	s.Log.Info("queue purged", "queue", s.QueueName)
	return nil
}

// Info returns summary attributes for the queue (approximate counts).
func (s *SQSService) Info(ctx context.Context) map[string]interface{} {
	ctx, cancel := context.WithTimeout(ctx, s.OperationTimeout)
	defer cancel()

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

// isFIFO returns true if queue name ends with .fifo
func isFIFO(name string) bool {
	return strings.HasSuffix(name, ".fifo")
}
