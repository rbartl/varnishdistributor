# Varnish Distributor

A lightweight HTTP request distributor that queries multiple Varnish endpoints and returns a JSON response with the status of all endpoints. Perfect for monitoring and load balancing across multiple Varnish instances.

## 🎯 Use Case

Varnish Distributor is designed for scenarios where you have multiple Varnish instances and need to:

- **Monitor health** of all Varnish endpoints from a single endpoint
- **Load balance** requests across multiple Varnish instances
- **Get comprehensive status** information for all endpoints in one request
- **Integrate with monitoring systems** that expect JSON responses

## ✨ Features

- **Multi-endpoint monitoring**: Query multiple Varnish instances simultaneously
- **JSON response format**: Structured response with status codes for all endpoints
- **Smart HTTP status codes**: Returns 200 only if all endpoints are healthy
- **Docker support**: Ready-to-use container images for easy deployment
- **Multi-architecture**: Supports both AMD64 and ARM64 platforms
- **Lightweight**: Minimal resource footprint

## 🚀 Quick Start

### Using Docker (Recommended)

```bash
# Pull the latest image
docker pull ghcr.io/rbartl/varnishdistributor:latest

# Run with your Varnish endpoints
docker run -p 6083:6083 ghcr.io/rbartl/varnishdistributor:latest \
  -a :6083 \
  varnish1.example.com:6081 \
  varnish2.example.com:6081 \
  varnish3.example.com:6081
```

### Using Binary

```bash
# Download the latest binary from GitHub releases
# Then run:
./varnishdistributor-linux-amd64 \
  -a :6083 \
  varnish1.example.com:6081 \
  varnish2.example.com:6081 \
  varnish3.example.com:6081
```

## 📋 Usage

### Command Line Options

```bash
varnishdistributor [OPTIONS] <varnish-endpoints...>

Options:
  -a, --address string    Listen address (default ":6083")

Arguments:
  varnish-endpoints       List of Varnish endpoints (host:port)
```

### Example Usage

```bash
# Basic usage
varnishdistributor -a :8080 varnish1:6081 varnish2:6081

# With custom port
varnishdistributor -a :9090 \
  varnish-prod-1.example.com:6081 \
  varnish-prod-2.example.com:6081 \
  varnish-staging.example.com:6081
```

## 📡 API Response

The distributor returns a JSON response with the status of all endpoints:

### Success Response (All endpoints healthy)
```json
{
  "endpoints": [
    {
      "server": "varnish1:6081",
      "status_code": 200,
      "status_text": "200 OK"
    },
    {
      "server": "varnish2:6081", 
      "status_code": 200,
      "status_text": "200 OK"
    }
  ],
  "all_ok": true
}
```

### Partial Failure Response
```json
{
  "endpoints": [
    {
      "server": "varnish1:6081",
      "status_code": 200,
      "status_text": "200 OK"
    },
    {
      "server": "varnish2:6081",
      "status_code": 503,
      "status_text": "503 Service Unavailable"
    },
    {
      "server": "varnish3:6081",
      "status_code": 0,
      "status_text": "Connection Error",
      "error": "connection refused"
    }
  ],
  "all_ok": false
}
```

### HTTP Status Codes

- **200 OK**: All endpoints are healthy
- **207 Multi-Status**: Some endpoints have issues (partial failure)

## 🐳 Docker Deployment

### Docker Compose Example

```yaml
version: '3.8'
services:
  varnish-distributor:
    image: ghcr.io/rbartl/varnishdistributor:latest
    ports:
      - "6083:6083"
    command: >
      -a :6083
      varnish-prod-1:6081
      varnish-prod-2:6081
      varnish-staging:6081
    restart: unless-stopped
```

### Kubernetes Example

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: varnish-distributor
spec:
  replicas: 2
  selector:
    matchLabels:
      app: varnish-distributor
  template:
    metadata:
      labels:
        app: varnish-distributor
    spec:
      containers:
      - name: distributor
        image: ghcr.io/rbartl/varnishdistributor:latest
        ports:
        - containerPort: 6083
        args:
        - -a
        - :6083
        - varnish-prod-1:6081
        - varnish-prod-2:6081
        - varnish-staging:6081
        livenessProbe:
          httpGet:
            path: /
            port: 6083
          initialDelaySeconds: 10
          periodSeconds: 30
---
apiVersion: v1
kind: Service
metadata:
  name: varnish-distributor-service
spec:
  selector:
    app: varnish-distributor
  ports:
  - port: 6083
    targetPort: 6083
  type: ClusterIP
```

## 🔧 Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/rbartl/varnishdistributor.git
cd varnishdistributor

# Install dependencies
go mod download

# Build the binary
go build -o varnishdistributor ./vdistribute.go

# Run locally
./varnishdistributor -a :6083 varnish1:6081 varnish2:6081
```

### Building Docker Image

```bash
# Build locally
docker build -t varnishdistributor .

# Run locally built image
docker run -p 6083:6083 varnishdistributor \
  -a :6083 varnish1:6081 varnish2:6081
```

## 📦 Releases

### Docker Images

Available on GitHub Container Registry:
- `ghcr.io/rbartl/varnishdistributor:latest`
- `ghcr.io/rbartl/varnishdistributor:v0.0.0` (versioned tags)

### Binary Downloads

Linux binaries are available in the [GitHub releases](https://github.com/rbartl/varnishdistributor/releases).

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 Support

If you encounter any issues or have questions:

1. Check the [GitHub Issues](https://github.com/rbartl/varnishdistributor/issues)
2. Create a new issue with detailed information about your problem
3. Include your Varnish configuration and the exact error messages

---

**Happy distributing! 🚀**
