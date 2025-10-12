package version

// These variables are intended to be overridden at build time via -ldflags.
// Example:
// go build -ldflags "-X github.com/pachecoc/sqs-ui/internal/version.Version=0.2.0 -X github.com/pachecoc/sqs-ui/internal/version.Commit=$(git rev-parse HEAD) -X github.com/pachecoc/sqs-ui/internal/version.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
var (
	Version   = "dev"
	Commit    = "none"
	BuildTime = "unknown"
)
