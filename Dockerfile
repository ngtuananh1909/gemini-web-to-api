# Stage 1: Build
FROM golang:1.25.1-alpine AS builder

WORKDIR /app

# Cache dependencies
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o main ./cmd/server/main.go

# Stage 2: Final Image
FROM alpine:3.22.2

# Install required packages
RUN apk add --no-cache ca-certificates tzdata

# Create a non-root user and group, and pre-create the persistent paths used by
# the Compose named volumes. The paths are owned before switching away from
# root so the gateway can write them as appuser.
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
RUN mkdir -p /home/appuser/.gateway /home/appuser/.cookies \
    && chown -R appuser:appgroup /home/appuser \
    && chmod 0700 /home/appuser/.gateway /home/appuser/.cookies

WORKDIR /home/appuser

ENV HOME=/home/appuser \
    GATEWAY_DATA_DIR=.gateway

# Copy file from builder and change ownership
COPY --from=builder --chown=appuser:appgroup /app/main .

# Switch to non-root user
USER appuser

EXPOSE 4981

ENTRYPOINT ["./main"] 
