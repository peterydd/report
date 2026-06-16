# Use official Go base image
FROM golang:1.26 as builder

# Enable Go modules
ENV GO111MODULE=on \
    GOPROXY=https://goproxy.io,direct

# Set working directory
WORKDIR /app

# Copy source code to container
COPY . .

# Build application
RUN cd ./cmd/report && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o app

# Use scratch as base image for minimal size
FROM scratch

# Copy built application from builder
COPY --from=builder /app/cmd/report/app /app

# Set entry point
ENTRYPOINT ["/app"]
