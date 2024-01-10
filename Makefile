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
	ko build --bare --platform=linux/amd64 ./cmd/bot

.PHONY: lint/golangci
lint/golangci:
	docker run --rm -v `pwd`:/app -w /app golangci/golangci-lint:v1.55.2 golangci-lint run -v --timeout=5m

.PHONY: lint/deps
lint/deps:
	go mod tidy
	go mod verify

lint: lint/deps lint/golangci