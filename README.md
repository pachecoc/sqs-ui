# AWS SQS Demo

A production-grade Go web app demonstrating AWS SQS integration using the AWS SDK v2, EKS IAM roles, and Helm.

## Features
- Send & fetch SQS messages (UI + REST)
- Works with `QUEUE_NAME` or `QUEUE_URL`
- Uses AWS SDK default credential chain
- Pretty Tailwind UI
- Logs all actions (`kubectl logs`)

## Deploy
```bash
helm install sqs-demo oci://ghcr.io/pachecoc/charts/sqs-demo --set aws.queueName=my-demo-queue
```
