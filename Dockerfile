FROM golang:1.21-alpine as builder

ENV CGO_ENABLED=0

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN go build -o /service/bot ./cmd/bot

FROM cgr.dev/chainguard/static

COPY --from=builder /service /service
WORKDIR /service

ENTRYPOINT ["/service/bot", "--telegram-debug=true", "--openai-debug=true"]
