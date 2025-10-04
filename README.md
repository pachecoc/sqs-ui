# 📬 SQS UI

A lightweight demo application to visualize and interact with **AWS SQS queues**.

It provides:
- Sending and receiving messages
- Viewing queue attributes
- Purging messages
- A simple Tailwind UI for quick testing

---

## 🧱 Project Structure

cmd/server - Application entrypoint
internal/config - Environment config loader
internal/service - SQS logic layer
internal/handler - REST API and routing
web/ - Static HTML UI


---

## 🚀 Running Locally

```bash
export QUEUE_NAME=my-queue
make run-local

Then open: http://localhost:8080
```

## Docker
```bash
docker buildx build --platform linux/amd64,linux/arm64 -t ghcr.io/pachecoc/sqs-ui:latest .
docker run -p 8080:8080 -e QUEUE_NAME=my-queue ghcr.io/pachecoc/sqs-ui:latest
```

## Notes

Works with AWS IAM roles, Pod Identity, or environment credentials.

Does not persist messages — reads directly from SQS.

Supports FIFO queues automatically.

## License

MIT © Gustavo Pacheco
