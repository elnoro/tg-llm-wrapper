include .env
export

.PHONY: run/local
run/local:
	go run cmd/bot/main.go --llm-engine=ollama --telegram-debug=true

.PHONY: run/openai
run/openai:
	go run cmd/bot/main.go --llm-engine=openai --telegram-debug=true --openai-debug=true

.PHONY: deploy
deploy:
	podman build -t tgllm-fly-deployer -f fly.Dockerfile .
	podman run --rm -it \
		-v $(shell pwd):/app \
		-e FLY_API_TOKEN=$(FLY_API_TOKEN) \
		tgllm-fly-deployer

.PHONY: lint/golangci
lint/golangci:
	podman run --rm -v `pwd`:/app -w /app golangci/golangci-lint:v1.55.2 golangci-lint run -v --timeout=5m

.PHONY: lint/deps
lint/deps:
	go mod tidy
	go mod verify

.PHONY: lint
lint: lint/deps lint/golangci

.PHONY: podman
podman/start:
	@podman build -t tg-llm-wrapper .
	@envsubst < pod.yaml | podman kube play --replace -
	@podman pod logs -f -c tg-llm-wrapper-pod-bot tg-llm-wrapper-pod

.PHONY: podman
podman/destroy:
	@podman pod stop tg-llm-wrapper-pod
	@podman pod rm tg-llm-wrapper-pod