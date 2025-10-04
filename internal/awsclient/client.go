package awsclient

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// extractRegionFromURL tries to extract the AWS region from an SQS queue URL.
// Example: https://sqs.eu-central-1.amazonaws.com/123456789012/my-queue
func extractRegionFromURL(queueURL string) (string, error) {
	if queueURL == "" {
		return "", fmt.Errorf("empty queue URL")
	}
	u, err := url.Parse(queueURL)
	if err != nil {
		return "", fmt.Errorf("invalid queue URL: %w", err)
	}
	re := regexp.MustCompile(`sqs\.([a-z0-9-]+)\.amazonaws\.com`)
	matches := re.FindStringSubmatch(u.Host)
	if len(matches) < 2 {
		return "", fmt.Errorf("could not extract region from queue URL")
	}
	return matches[1], nil
}

func NewSQSClient(ctx context.Context, queueURL string) *sqs.Client {
	// Load default AWS configuration (uses env vars, EC2/EKS metadata, etc.)
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		log.Fatalf("failed to load AWS config: %v", err)
	}

	// If SDK couldn't determine region, try parsing it from queue URL
	if cfg.Region == "" && queueURL != "" {
		if region, err := extractRegionFromURL(queueURL); err == nil {
			cfg.Region = region
			log.Printf("✅ Inferred region from queue URL: %s", region)
		} else {
			log.Printf("⚠️ Could not infer region from queue URL: %v", err)
		}
	}

	if cfg.Region == "" {
		log.Fatalf("no AWS region found (not provided, not auto-detected, and not in queue URL)")
	}

	return sqs.NewFromConfig(cfg, func(o *sqs.Options) {
		o.Region = cfg.Region
	})
}
