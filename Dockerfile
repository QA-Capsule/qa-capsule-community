FROM golang:alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
ENV CGO_ENABLED=1
COPY . .
RUN go mod tidy
RUN go build -o qacapsule ./cmd/qacapsule/main.go

FROM alpine:latest
RUN apk add --no-cache bash curl jq ca-certificates

RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" \
    && chmod +x kubectl && mv kubectl /usr/local/bin/

WORKDIR /app
COPY --from=builder /app/qacapsule .
COPY config.yaml .
COPY web/ ./web/

EXPOSE 9000
CMD ["./qacapsule"]