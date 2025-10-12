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
}

// SQSService wraps SQS operations with configuration and logging.
type SQSService struct {
	Client                   *sqs.Client
	QueueName                string
	QueueURL                 string
	Region                   string
	Log                      *slog.Logger
}

func NewSQSService(ctx context.Context, client *sqs.Client, queueName, queueURL, region string, log *slog.Logger) *SQSService {
	log.Debug("creating SQS service", "queue_name", queueName, "queue_url", queueURL)

	s := &SQSService{
		Client:                   client,
		QueueName:                queueName,
		QueueURL:                 queueURL,
		Region:                   region,
		Log:                      log,
	}

	// If queue URL is provided, extract name
	if queueURL != "" {
		parts := strings.Split(queueURL, "/")
		s.QueueName = parts[len(parts)-1]
		log.Info("extracted queue name from URL", "queue_name", s.QueueName)
	}

	return s
}

// EnsureQueueConfigured verifies that either QueueName or QueueURL is set.
// Exported so handlers can call it before performing queue-dependent actions.
func (s *SQSService) EnsureQueueConfigured() error {
    if s.QueueURL == "" && s.QueueName == "" {
        if s.Log != nil {
            s.Log.Info("no queue configured")
        }
        return fmt.Errorf("please set either queue name or queue URL before performing this action")
    }
    return nil
}

// FetchQueueURL attempts to resolve the queue URL from AWS using the queue name.
func (s *SQSService) FetchQueueURL(ctx context.Context) (string, error) {
	s.Log.Debug("fetching queue URL", "queue_name", s.QueueName)

	if s.Client == nil {
		return "", fmt.Errorf("no AWS client configured")
	}

	resolveCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	resp, err := s.Client.GetQueueUrl(resolveCtx, &sqs.GetQueueUrlInput{
		QueueName: &s.QueueName,
	})
	if err != nil {
		s.Log.Warn("failed to resolve queue URL", "queue_name", s.QueueName, "error", err)
		return "", err
	}

	s.QueueURL = *resp.QueueUrl
	s.Log.Info("resolved queue URL", "queue_name", s.QueueName, "queue_url", s.QueueURL)

	return s.QueueURL, nil
}

// Send publishes a message to the queue (adds group id if FIFO).
func (s *SQSService) Send(ctx context.Context, msg string) error {
	s.Log.Debug("sending message", "msg", msg)

    if s.QueueURL == "" {
        s.Log.Warn("send skipped — no active queue configured")
        return fmt.Errorf("no active queue configured, try to fetch queue info first")
    }
    if strings.TrimSpace(msg) == "" {
        return fmt.Errorf("message body cannot be empty")
    }

	// Set a timeout for each operation
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	input := &sqs.SendMessageInput{
		QueueUrl:    &s.QueueURL,
		MessageBody: &msg,
	}

	// If FIFO queue, set MessageGroupId and ensure a MessageDeduplicationId
    if isFIFO(s.QueueURL) {
        groupID := "default-group"
        input.MessageGroupId = &groupID

        // If your queue does not enable content-based deduplication, always set a deduplication id.
        dedupID := fmt.Sprintf("%d-%s", time.Now().UnixNano(), s.QueueName)
        input.MessageDeduplicationId = &dedupID
    }

	if _, err := s.Client.SendMessage(ctx, input); err != nil {
		return fmt.Errorf("failed to send message: %w", err)
	}

	s.Log.Info("message sent", "queueName", s.QueueName, "queueURL", s.QueueURL)

	// TODO: return info?
	return nil
}

// ReceiveAll behaves in loop mode regardless,
// aggregating batches until an empty batch, iteration cap, or timeout occurs.
func (s *SQSService) ReceiveAll(ctx context.Context, max int32) ([]map[string]interface{}, error) {
	s.Log.Debug("receiving messages", "max", max)

	if s.QueueURL == "" {
		s.Log.Info("receive skipped — no active queue configured")
		return nil, fmt.Errorf("no active queue configured, try to fetch queue info first")
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
	receiveLoop:
		for iteration := 1; iteration <= maxIterations; iteration++ {
			select {
			case <-ctx.Done():
				if len(allMsgs) > 0 {
					s.Log.Warn("receiveAll cancelled after partial retrieval", "count", len(allMsgs))
					break receiveLoop
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

	elapsed := time.Since(start)
	s.Log.Info("messages fetched",
		"count", len(allMsgs),
		"elapsed_ms", elapsed.Milliseconds(),
	)

	s.Log.Info("receiveAll completed", "total_messages", len(allMsgs), "elapsed_ms", elapsed.Milliseconds())
	return allMsgs, nil
}

// Purge deletes all messages currently in the queue.
func (s *SQSService) Purge(ctx context.Context) error {
	s.Log.Debug("purging queue", "queue", s.QueueName)

	if s.QueueURL == "" {
		s.Log.Info("purge skipped — no active queue configured")
		return fmt.Errorf("no active queue configured, try to fetch queue info first")
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
	s.Log.Debug("fetching queue info", "queue_name", s.QueueName, "queue_url", s.QueueURL)

	// Base info map
	info := map[string]interface{}{
		"current_region": s.Region,
		"queue_name":    s.QueueName,
		"queue_url":      s.QueueURL,
		"number_of_messages": nil,
		"status":         "not_connected",
	}

	// Ensure the queue is configured before fetching info
	if err := s.EnsureQueueConfigured(); err != nil {
		s.Log.Info("queue is not configured", "error", err)
		info["error"] = err.Error()
		return info
	}

	// If no URL, we should fetch it with the name
	if s.QueueURL == "" && s.QueueName != "" {
		queueURL, err := s.FetchQueueURL(ctx)
		if err != nil {
			s.Log.Info("queue could not be loaded — running in idle mode", "queue_name", s.QueueName)
			info["error"] = err.Error()
			return info
		}
		info["queue_url"] = queueURL
	}

	// Once we have a URL, we can fetch attributes with a timeout
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	out, err := s.Client.GetQueueAttributes(ctx, &sqs.GetQueueAttributesInput{
		QueueUrl: &s.QueueURL,
		AttributeNames: []types.QueueAttributeName{
			types.QueueAttributeNameApproximateNumberOfMessages,
			types.QueueAttributeNameApproximateNumberOfMessagesNotVisible,
			types.QueueAttributeNameApproximateNumberOfMessagesDelayed,
		},
	})
	if err != nil {
		s.Log.Warn("failed to get queue attributes", "error", err)
		info["error"] = err.Error()
		return info
	}

	// Parse attributes safely
	parseInt := func(v string) int64 {
		n, _ := strconv.ParseInt(v, 10, 64)
		return n
	}

	visible := parseInt(out.Attributes[string(types.QueueAttributeNameApproximateNumberOfMessages)])
	notVisible := parseInt(out.Attributes[string(types.QueueAttributeNameApproximateNumberOfMessagesNotVisible)])
	delayed := parseInt(out.Attributes[string(types.QueueAttributeNameApproximateNumberOfMessagesDelayed)])

	info["approximate_number_of_messages"] = visible
	info["approximate_number_of_messages_not_visible"] = notVisible
	info["approximate_number_of_messages_delayed"] = delayed

	info["number_of_messages"] = strconv.FormatInt(visible+notVisible+delayed, 10)
	info["status"] = "ok"

	s.Log.Info("queue info fetched", "queue_name", s.QueueName, "queue_url", s.QueueURL)
	s.Log.Debug("queue info details", "info", info)
	return info
}

// isFIFO returns true if queue name contains fifo
func isFIFO(name string) bool {
	slog.Debug("checking if FIFO", "queue_name", name)
	return strings.Contains(name, "fifo")
}
