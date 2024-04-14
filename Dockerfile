# Start from Ubuntu 20.04 as the base for the build stage to ensure compatibility
FROM golang:1.21.0 as builder

WORKDIR /app

# Copy your source code
COPY . .

# Build your Go application
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags '-extldflags "-static"' -v -o /fly/bin/start ./cmd/start
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags '-extldflags "-static"' -v -o /fly/bin/cm ./cmd/cm

COPY ./bin/* /fly/bin/
COPY ./schedules.json /fly/schedules.json

# Start from Ubuntu 20.04 for the runtime stage
FROM ubuntu:22.04

# Avoid debconf warnings during the build
ENV DEBIAN_FRONTEND=noninteractive

# Install runtime dependencies
RUN apt-get update && \
    apt-get install -y --no-install-recommends libsqlite3-dev sqlite3 cron curl ca-certificates && \
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Copy the built binary from the builder stage
COPY --from=builder /fly/bin/* /usr/local/bin/
COPY --from=builder /fly/schedules.json /usr/local/share/

# Set the CMD to your application
CMD ["start"]
