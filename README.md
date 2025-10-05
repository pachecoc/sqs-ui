# 📬 SQS UI

A lightweight demo application to visualize and interact with **AWS SQS queues**.

**Features:**
- Send and receive messages (peek, non-destructive)
- View queue attributes (message counts, status)
- Purge all messages in the queue
- Simple, responsive Tailwind UI for quick testing
- Supports both Standard and FIFO queues

---

## 🧱 Project Structure

- `cmd/server` — Application entrypoint
- `internal/settings` — Environment config loader
- `internal/service` — SQS logic layer
- `internal/handler` — REST API and routing
- `internal/logging` — Structured logger
- `web/` — Static HTML UI

---

## 🚀 Running Locally

```bash
export QUEUE_NAME=my-queue
make run-local
# Then open: http://localhost:8080
```

---

## 🐳 Docker

```bash
docker run -p 8080:8080 \
    -e QUEUE_NAME=my-queue \
    -e AWS_ACCESS_KEY_ID=... \
    -e AWS_SECRET_ACCESS_KEY=... \
    ghcr.io/pachecoc/sqs-ui:latest
```

---

## ⚙️ Configuration

Set these environment variables to configure the app:

| Variable        | Description                                                                              | Default |
| --------------- | ---------------------------------------------------------------------------------------- | ------- |
| `QUEUE_NAME`    | SQS queue name (required if no URL)                                                      |         |
| `QUEUE_URL`     | SQS queue URL (optional, overrides name)                                                 |         |
| `PORT`          | HTTP port to listen on                                                                   | `8080`  |
| `LOG_LEVEL`     | Logging level (`debug`, `info`, `warn`, `error`)                                         | `info`  |
| AWS credentials | Standard AWS env vars (`AWS_REGION`, `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, etc.) |         |

- Either `QUEUE_NAME` or `QUEUE_URL` **must** be set.
- AWS credentials can be provided via environment, IAM role, or Pod Identity.

---

## 📝 Notes

- Reads directly from SQS; does **not** persist messages.
- FIFO queues are detected automatically and handled.
- All message fetches are non-destructive (peek only).
- Purge deletes all messages in the queue.
- Compatible with AWS IAM roles, Pod Identity, or environment credentials.

---

## 🧑‍💻 Development

- Build locally: `make build-local`
- Run tests: *(not implemented)*
- Tidy modules: `make tidy`
- Multi-arch Docker build: `make build`

---

## 📄 License

MIT © Gustavo Pacheco
