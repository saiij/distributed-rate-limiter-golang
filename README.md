# Distributed Rate Limiter

A distributed rate limiter built in Go, using Redis as the shared state backend to enforce request limits consistently across multiple service instances.

## Architecture

```
Client → Nginx (load balancer)
              ↓
    ┌─────────────────┐
    │  App Instance 1 │
    │  App Instance 2 │  → Redis (shared state)
    │  App Instance 3 │
    └─────────────────┘
```

- **Multiple service instances** share state through Redis
- **Lua scripts** ensure atomic operations — no race conditions across instances
- **Clean interfaces** allow swapping algorithms or storage backends without changing business logic

## Algorithms

| Algorithm | Description |
|---|---|
| **Fixed Window** | Counts requests in fixed time windows. Simple and fast. |
| **Sliding Window** | Weighted combination of current and previous window. Prevents burst exploitation at window edges. |
| **Token Bucket** | Bucket of tokens refilled at a constant rate. Allows controlled bursts. |

## Running locally

**Requirements:** Docker and Docker Compose

```bash
# Start 3 app instances + Redis + Nginx
docker compose up --scale app=3 -d

# Check service is running
curl http://localhost:8080/health
```

## API

### `POST /check`

Check if a request is allowed for a given key.

**Request:**
```json
{
  "key": "ip:192.168.1.1"
}
```

**Response:**
```json
{
  "can_access": true,
  "request_remain": 99,
  "retry_in": "1m0s",
  "reset_request_at": "Wed, 08 Apr 2026 01:07:28 GMT"
}
```

**Status codes:**
- `200 OK` — request allowed
- `429 Too Many Requests` — rate limit exceeded
- `400 Bad Request` — missing or invalid body

### `GET /health`

```json
{ "status": "ok" }
```

## Testing the rate limit

```bash
for i in $(seq 1 105); do
  echo -n "Request $i: "
  curl -s -X POST http://localhost:8080/check \
    -H "Content-Type: application/json" \
    -d '{"key": "ip:127.0.0.1"}'
  echo
done
```

Requests 1-100 return `200`, requests 101+ return `429`.
