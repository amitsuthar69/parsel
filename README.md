## Parsel DLASS

Parsel aggregates logs from files, transports them through a Redis Stream, and fans them out in real time to multiple independent consumers.

### Proposed System Architecture:

<img width="3826" height="1661" alt="image" src="https://github.com/user-attachments/assets/df929845-14f7-4fd6-babf-3d807a1c4f8d" />

### How it works

**Agent** watches a directory for `*.log`, when a file is written to, it reads the new bytes, parses each line as JSON *containerd-format*, and pushes a structured log entry to a Redis Stream.

**Redis Stream** acts as the transport layer. Each egress component has its own consumer group, so they all receive every message independently and track their own progress.

**Logger consumer** (demo) reads from the stream and prints every log entry to stdout. (todo: replace it with a persistent DB writer)

**Alerter consumer** reads from the stream and prints a formatted alert for every `ERROR`-level log.

**WebSocket gateway** reads from the stream and fans out every log entry in real time to all connected WebSocket clients.

---

### Quickstart

**Prerequisites:** Docker and Docker Compose

```bash
git clone https://github.com/amitsuthar69/parsel
cd parsel
docker compose up --build
```

To see logs streaming over WebSocket:

```bash
websocat ws://localhost:8080/ws
```

### Log format

Parsel expects logs in containerd JSON format, one JSON object per line:

```json
{"log":"your message here","stream":"stdout","time":"2026-06-19T10:00:00Z"}
```

The service name is derived from the log filename. A file named `payment.log` produces logs with `"service": "payment"`.

### Configuration

All components are configured via environment variables.

| Variable | Default | Description |
|---|---|---|
| `REDIS_ADDR` | `localhost:6379` | Redis address |
| `STREAM_NAME` | `parsel:logs` | Redis Stream key |
| `LOG_DIR` | `/var/log/containers` | Directory the agent watches |
| `WS_ADDR` | `:8080` | WebSocket gateway listen address |
| `NODE_NAME` | hostname | Identifier for this node |

---

### Project structure

```
parsel/
├── cmd/
│   ├── agent/        ← log file watcher and Redis publisher
│   ├── alerter/      ← ERROR log consumer
│   ├── consumer/     ← demo consumer (logs to stdout)
│   ├── producer/     ← demo log file writer
│   └── wsgateway/    ← WebSocket fan-out gateway
├── internal/
│   ├── config/
│   ├── consumer/     ← shared consumer logic
│   └── models/
├── Dockerfile
├── docker-compose.yml
└── .env
```
