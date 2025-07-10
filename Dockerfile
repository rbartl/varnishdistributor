# Build stage
FROM golang:1.21-alpine AS builder

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o varnishdistributor ./vdistribute.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /opt/

# Copy the binary from builder stage
COPY --from=builder /app/varnishdistributor .

# Expose port
EXPOSE 6083

# Run the binary
ENTRYPOINT ["./varnishdistributor"]

