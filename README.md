# JINA HTTP Proxy

A proxy service that automatically adds API keys to outgoing HTTP requests. It provides functionality for key management and rotation.

## Features

- HTTP Proxy that adds API keys to requests
- Key management API for adding keys and viewing statistics
- Automatic key rotation using a "best key" algorithm
- Database persistence for keys using PostgreSQL

## Prerequisites

- Docker and Docker Compose
- Go 1.24+ (for local development)

## Running with Docker Compose

The easiest way to get started is using Docker Compose:

```bash
# Build and start the containers
docker-compose up -d

# To stop the containers
docker-compose down
```

The services will be available at:

- Proxy Server: <http://localhost:5555>
- API Server: <http://localhost:5556>

## HTTP Proxy

Must be add proxy CA to your system trust store. Check `./ca.crt` or `./ca.pem` for CA certificate.

Example:

```bash
curl --location 'https://api.jina.ai/v1/embeddings' \
   --header 'Content-Type: application/json' \
   --data '{
   "model": "jina-clip-v2",
   "input": [
      {
            "text": "A beautiful sunset over the beach"
      }
   ]
   }' -x http://localhost:5555
```

## API Endpoints

### Insert a new API key

```bash
curl -X POST http://localhost:5556/keys -H "Content-Type: application/json" -d '{"key":"your-api-key"}'
```

### Get Key Statistics

```bash
curl http://localhost:5556/keys/stats
```

Response:

```json
{
  "Count": 5,
  "Balance": 5000000
}
```

## Environment Variables

- `GOOSE_DBSTRING`: PostgreSQL connection string (required)
- `GOOSE_MIGRATION_DIR`: Path to the database migration files (required)

## Development

### Prerequisites

- Go 1.24+
- PostgreSQL

### Setup

1. Clone the repository:

   ```bash
   git clone https://github.com/trancong12102/jina-http-proxy.git
   cd jina-http-proxy
   ```

2. Install dependencies:

   ```bash
   go mod download
   ```

3. Create a `.env` file with the following content:

```bash
   GOOSE_DBSTRING=postgres://postgres:postgres@localhost:5432/jina_proxy?sslmode=disable
   GOOSE_MIGRATION_DIR=./migrations
```

4. Run the application:

   ```bash
   go run main.go
   ```

## Building for Production

To build the application binary:

```bash
go build -o jina-http-proxy .
```

## License

[MIT License](LICENSE)
