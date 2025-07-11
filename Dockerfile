# Build stage
FROM golang:1.24.5-alpine AS builder

WORKDIR /app

# Install git for go mod download if needed
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -v -ldflags="-s -w" -o mailhole ./cmd/mailhole

# Final image
FROM alpine:3.22

WORKDIR /app

# Copy the binary from the builder
COPY --from=builder /app/mailhole .

# Optionally copy migrations or config if needed
# COPY db/migrations ./db/migrations

# Expose ports (adjust as needed)
EXPOSE 80 25

# Set environment variables (optional)
ENV DB_URL=postgres://mailhole:mailhole@localhost:5432/mailhole?sslmode=disable
ENV SMTP_ADDR=:25
ENV HTTP_ADDR=:80

# Run the app
ENTRYPOINT ["./mailhole"]
