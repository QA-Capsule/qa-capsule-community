FROM golang:alpine AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
ENV CGO_ENABLED=1
COPY . .
RUN go mod tidy
# OPTIMIZATION: Use ldflags -s and -w to strip debug information and reduce binary size.
RUN go build -ldflags="-s -w" -o qacapsule ./cmd/qacapsule/main.go

FROM alpine:latest
RUN apk add --no-cache bash curl jq ca-certificates

RUN curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" \
    && chmod +x kubectl && mv kubectl /usr/local/bin/

# SECURITY: Create a non-root system user and group to run the application.
RUN addgroup -S appgroup && adduser -S appuser -G appgroup

WORKDIR /app
COPY --from=builder /app/qacapsule .
COPY config.yaml .
COPY web/ ./web/

# SECURITY: Ensure the data directory exists and belongs to the non-root user for SQLite DB writes.
RUN mkdir -p ./data && chown -R appuser:appgroup /app

# SECURITY: Switch from root to the non-privileged user before executing the binary.
USER appuser

EXPOSE 9000
CMD ["./qacapsule"]