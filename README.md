# 📬 SQS UI

A production-ready Go web application that connects to **AWS SQS** using the AWS SDK v2.
It provides both a **simple web UI** and a **REST API** to send and receive messages from your queue.

Designed for EKS environments using **Pod Identity / IRSA**,
but it also works locally with standard AWS credentials.

---

## 🚀 Features

- Uses the **AWS SDK v2** with the default credential provider chain:
  - EKS Pod Identity / IRSA
  - Environment variables
  - Shared credentials / EC2 metadata
- Works with either:
  - `QUEUE_NAME` → resolves automatically via `GetQueueUrl`
  - `QUEUE_URL` → directly connects to a specific queue
- Beautiful, responsive web UI built with **TailwindCSS**
- REST API endpoints (`/send`, `/messages`, `/info`)
- Structured JSON logging via `log/slog`
- Runs as a minimal, secure **distroless** Docker image

---

## 🧩 Environment Variables

| Variable     | Required | Description                               |
| ------------ | -------- | ----------------------------------------- |
| `QUEUE_NAME` | optional | Name of the SQS queue (auto-resolves URL) |
| `QUEUE_URL`  | optional | Full SQS queue URL                        |
| `PORT`       | optional | HTTP port (default: 8080)                 |

> Either `QUEUE_NAME` or `QUEUE_URL` **must** be set.

---

## 🧰 Run Locally

Make sure your AWS credentials are configured (via `~/.aws/credentials` or env vars).

```bash
go run ./cmd/server

docker build -t sqs-ui:latest .
docker run -p 8080:8080 \
  -e QUEUE_NAME=my-ui-queue \
  -e AWS_REGION=eu-central-1 \
  -v ~/.aws:/root/.aws:ro \
  sqs-ui:latest
```
