build:
	docker build -t ghcr.io/pachecoc/sqs-ui:latest .

push:
	docker push ghcr.io/pachecoc/sqs-ui:latest

run-local:
	docker run -p 8080:8080 -e QUEUE_NAME=my-demo-queue ghcr.io/pachecoc/sqs-ui:latest
