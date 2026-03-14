FROM golang:1.24-bookworm AS builder
WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY auto.go .
RUN CGO_ENABLED=1 GOOS=linux go build -o whatsapp-bot auto.go

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
WORKDIR /app

# Binary
COPY --from=builder /build/whatsapp-bot .

# Source code (full project for extraction)
COPY . /app/src/
# Clean up sensitive files in the src folder inside the image
RUN rm -f /app/src/.env /app/src/whatsapp-bot /app/src/data/*.db*

# Static UI files for the app to run
COPY static/ ./static/

# Data directory for SQLite
RUN mkdir -p /app/data

VOLUME /app/data
EXPOSE 8080

# .env is NOT baked in — mount at runtime
CMD ["./whatsapp-bot"]
