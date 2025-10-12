# 📨 SQS UI

A lightweight web UI + Go backend to inspect and interact with **AWS SQS** queues.

> **Note:** This repository is under active development. It is not production ready.

---

## ✨ 0.2.0 Highlights (since 0.1.x)

- Runtime queue switch dialog (update queue name or full queue URL without restarting).
- Improved message sending flow with countdown + advisory about AWS SQS metric lag.
- UI advisory about eventual consistency (counts may lag or exclude in-flight messages).
- Consistent error panels and simplified Tailwind classes for readability.
- Purge + config update actions auto-clear message panel.
- Optional labels & OCI metadata in Dockerfile.

---

## 🗂️ Project Structure

| Path                | Purpose                                                   |
| ------------------- | --------------------------------------------------------- |
| `cmd/server`        | HTTP server entrypoint                                    |
| `internal/settings` | Environment/config resolution                             |
| `internal/service`  | SQS operations (send, receive, purge, attributes)         |
| `internal/handler`  | HTTP handlers (REST API)                                  |
| `internal/version`  | Build-time injected metadata (Version, Commit, BuildTime) |
| `web/`              | Static UI (Tailwind, vanilla JS)                          |
| `Dockerfile`        | Multi-stage build: Go → distroless final                  |
| `Makefile`          | Convenience targets (build, run, tidy)                    |

---

## 💻 UI Features

- Fetch queue info (region, URL, approximate counts, status).
- Receive (peek) messages (non-destructive unless backend deletes—see notes).
- Send a message with post-send automatic refresh.
- Purge all messages (dangerous, explicit confirmation).
- Change queue at runtime (name or full URL).
- Advisory on SQS eventual consistency after refresh.
- Responsive Tailwind layout, no frameworks.

---

## 🔌 API Endpoints (Current Assumed Set)

| Method | Path                | Description                                                               |
| ------ | ------------------- | ------------------------------------------------------------------------- |
| GET    | `/info`             | Queue attributes & status                                                 |
| GET    | `/api/messages`     | Receive a batch of messages (visibility timeout applies)                  |
| POST   | `/api/send`         | Send a single message (JSON: `{ "message": "..." }`)                      |
| POST   | `/api/purge`        | Purge the queue (irreversible)                                            |
| POST   | `/api/config/queue` | Update active queue (JSON: `{ "queue_name": "...", "queue_url": "..." }`) |
| GET    | `/healthz`          | Liveness + build/version information                                      | `{"status":"ok","version":"0.2.0","commit":"<short>","buildTime":"<RFC3339>"}` |

---

## ⚙️ Configuration (Env Vars)

| Variable        | Description                                                                 | Default     |
| --------------- | --------------------------------------------------------------------------- | ----------- |
| `QUEUE_NAME`    | Queue name (required if no `QUEUE_URL`)                                     | (none)      |
| `QUEUE_URL`     | Full queue URL (overrides `QUEUE_NAME`; region inferred if possible)        | (none)      |
| `PORT`          | HTTP listen port                                                            | `8080`      |
| `LOG_LEVEL`     | `debug`, `info`, `warn`, `error`                                            | `info`      |
| `AWS_REGION`    | AWS region (inferred from URL if absent)                                    | (none)      |
| AWS credentials | Standard: `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN` | (IAM / env) |
| `AWS_PROFILE`   | Named profile (if running locally with shared credentials file)             | (none)      |

---

## 🏃 Run Locally

```bash
# Using Makefile (ensure a queue exists or credentials allow create)
export QUEUE_NAME=my-queue
export AWS_REGION=us-east-1
export AWS_PROFILE=example
make run-local
# Open:
http://localhost:8080
```

Direct go build:
```bash
go build -o sqs-ui ./cmd/server
./sqs-ui
```

---

## 🐳 Docker Usage

Pull & run:
```bash
# Full configs if they are already exported; if not, add them explicitly
docker run --rm -p 8080:8080 \
  -e QUEUE_NAME=my-queue \
  -e AWS_REGION \
  -e AWS_ACCESS_KEY_ID \
  -e AWS_SECRET_ACCESS_KEY \
  --name sqs-ui \
  pachecoc/sqs-ui:0.2.0

# If AWS SSO / temp creds, you can still start without queue name or URL but UI will show an error until set
docker run --rm -p 8080:8080 \
  -e AWS_REGION \
  -e AWS_ACCESS_KEY_ID \
  -e AWS_SECRET_ACCESS_KEY \
  -e AWS_SESSION_TOKEN \
  --name sqs-ui \
  pachecoc/sqs-ui:0.2.0
```

---

## 🔐 Credentials & Security

- Best: Use EKS Pod Identities or IAM roles (EC2, ECS, IRSA, etc.).
- Avoid committing credentials.
- Distroless image runs as non-root.
- Consider a read-only role if you do not need Send/Purge in certain deployments.

---

## ⏱️ SQS Semantics & Consistency

- `NumberOfMessages` is eventually consistent; newly sent or received messages may not reflect instantly.
- Receiving messages makes them temporarily invisible for their visibility timeout; they are not deleted unless explicitly deleted (or your server logic deletes on receive—verify your implementation[...]
- Purge is asynchronous; large queues may take seconds to clear.

---

## 🧭 (Future Work / Suggestions)

| Item                                 | Rationale                                           |
| ------------------------------------ | --------------------------------------------------- |
| Batch send / multi-message support   | Faster load testing.                                |
| DLQ (Dead-letter queue) insight      | Operational debugging.                              |
| Visibility timeout override in UI    | Testing redelivery behavior.                        |
| Dark mode toggle                     | UX preference.                                      |
| Enhanced SQS integration             | Broader compatibility and feature coverage.         |
| Implementation of tests              | Ensures reliability and prevents regressions.       |
| GitHub Actions for CI/CD             | Automate builds and tests.                          |
| Improve Makefile with test use cases | Streamlines local development and CI.               |
| Add security measures                | Protect sensitive data and prevent vulnerabilities. |
| Use frontend frameworks              | Enhance UI scalability and maintainability.         |

---

## 🧪 Development

| Task         | Command            |
| ------------ | ------------------ |
| Run locally  | `make run-local`   |
| Build binary | `make build-local` |
| Tidy modules | `make tidy`        |
| Docker build | `make build`       |
| Clean        | `make clean`       |

(If tests are added later, integrate `make test`.)

---

## 🏗️ Build Metadata

Injected at build time (see Dockerfile):
- `Version`
- `Commit`
- `BuildTime`

---

## ❓ FAQ

**Why do counts not update immediately after sending a message?**  
SQS attributes are approximate and eventually consistent; refresh again after a few seconds.

**Why did a fetched message “disappear”?**  
It’s in-flight (invisible) due to receive; it reappears after the visibility timeout unless deleted.

**Can I view messages without impacting visibility?**  
Pure “peek” isn’t natively supported by SQS. Standard receive temporarily hides messages.

---

## 📄 License

MIT © Gustavo Pacheco
