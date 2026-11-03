<div align="center">

<img src="https://raw.githubusercontent.com/devicons/devicon/master/icons/go/go-original.svg" alt="Go Logo" width="100" height="100" />

# engage-engine

**High-Throughput Social Engagement Engine — Distributed Systems Architecture in Go**

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=for-the-badge&logo=go&logoColor=white)](https://go.dev/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-4169E1?style=for-the-badge&logo=postgresql&logoColor=white)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7.x-DC382D?style=for-the-badge&logo=redis&logoColor=white)](https://redis.io/)
[![Apache Kafka](https://img.shields.io/badge/Apache_Kafka-3.x-231F20?style=for-the-badge&logo=apachekafka&logoColor=white)](https://kafka.apache.org/)
[![Docker](https://img.shields.io/badge/Docker-Containerized-2496ED?style=for-the-badge&logo=docker&logoColor=white)](https://www.docker.com/)
[![New Relic](https://img.shields.io/badge/New_Relic-APM_Observability-1CE783?style=for-the-badge&logo=newrelic&logoColor=black)](https://newrelic.com/)

<p align="center">
  <img src="https://skillicons.dev/icons?i=go,postgres,redis,kafka,docker,nginx" alt="Tech Stack" />
</p>

</div>

---

`engage-engine` is a scalable, production-oriented Go backend structured as a **Feature-Based Modular Monolith** combined with **Clean Architecture** principles. Designed to effortlessly survive massive viral engagement surges, it integrates **Redis HyperLogLog** for fixed-memory (12KB) cardinality deduplication, **Kafka micro-batching** for lock-free database write-behind persistence, and **probabilistic Cuckoo filters** for sub-millisecond membership checks.

---

## 🛠️ Tech Stack & Key Libraries

| Technology / Library | Purpose & Implementation Details | Logo / Badge |
| :--- | :--- | :---: |
| **Go (Golang)** | High-concurrency core runtime using standard library `net/http` router | <img src="https://raw.githubusercontent.com/devicons/devicon/master/icons/go/go-original.svg" width="32" height="32" alt="Go"/> |
| **PostgreSQL (`pgx/v5`)** | ACID datastore with high-performance connection pooling (`puddle`) and raw SQL | <img src="https://raw.githubusercontent.com/devicons/devicon/master/icons/postgresql/postgresql-original.svg" width="32" height="32" alt="PostgreSQL"/> |
| **Redis (`go-redis/v9`)** | HyperLogLog (`PFADD`/`PFCOUNT`), pipelined atomic counters, and cache | <img src="https://raw.githubusercontent.com/devicons/devicon/master/icons/redis/redis-original.svg" width="32" height="32" alt="Redis"/> |
| **Apache Kafka (`kafka-go`)** | Distributed event streaming (`engage.views`, `engage.follows`, `engage.likes`) | <img src="https://raw.githubusercontent.com/devicons/devicon/master/icons/apachekafka/apachekafka-original.svg" width="32" height="32" alt="Kafka"/> |
| **New Relic (`go-agent/v3`)** | Distributed tracing & APM profiling with `nrpgx5` and `nrredis-v9` datastore hooks | <img src="https://img.shields.io/badge/New_Relic-go--agent-1CE783?style=flat-square&logo=newrelic&logoColor=black" alt="New Relic"/> |
| **Zerolog (`rs/zerolog`)** | High-performance, zero-allocation structured JSON logging | <img src="https://img.shields.io/badge/zerolog-structured_json-blue?style=flat-square" alt="Zerolog"/> |
| **Validator (`validator/v10`)** | Declarative request payload & struct constraint validation | <img src="https://img.shields.io/badge/validator-v10-green?style=flat-square" alt="Validator"/> |
| **JWT (`golang-jwt/v5`)** | Secure stateless authentication & token claim validation | <img src="https://img.shields.io/badge/JWT-v5-black?style=flat-square&logo=jsonwebtokens" alt="JWT"/> |
| **Docker & Docker Compose** | Multi-stage Docker builds and automated local development orchestration | <img src="https://raw.githubusercontent.com/devicons/devicon/master/icons/docker/docker-original.svg" width="32" height="32" alt="Docker"/> |

---

## 1. Project Structure

```text
engage-engine/
├── cmd/
│   ├── api/
│   │   └── main.go               # HTTP API server entry point
│   └── worker/
│       └── main.go               # Background worker / event consumer entry point
│
├── internal/
│   ├── modules/                  # Independent feature modules (Clean Architecture)
│   │   ├── user/
│   │   │   ├── domain/
│   │   │   ├── usecase/
│   │   │   ├── repository/
│   │   │   └── handler/
│   │   ├── post/
│   │   │   ├── domain/
│   │   │   ├── usecase/
│   │   │   ├── repository/
│   │   │   └── handler/
│   │   ├── like/
│   │   │   ├── domain/
│   │   │   ├── usecase/
│   │   │   ├── repository/
│   │   │   └── handler/
│   │   ├── comment/
│   │   │   ├── domain/
│   │   │   ├── usecase/
│   │   │   ├── repository/
│   │   │   └── handler/
│   │   ├── follow/
│   │   │   ├── domain/
│   │   │   ├── usecase/
│   │   │   ├── repository/
│   │   │   └── handler/
│   │   ├── notification/
│   │   │   ├── domain/
│   │   │   ├── usecase/
│   │   │   ├── repository/
│   │   │   └── handler/
│   │   └── views/
│   │       ├── domain/
│   │       ├── usecase/
│   │       ├── repository/
│   │       └── handler/
│   │
│   ├── platform/                 # Infrastructure & technology adapters
│   │   ├── mongodb/              # Primary document database
│   │   ├── postgres/             # Optional relational database (pgxpool)
│   │   ├── redis/                # Caching & HyperLogLog
│   │   ├── kafka/                # Asynchronous event streaming
│   │   ├── logger/               # Structured logging (zerolog)
│   │   ├── config/               # Configuration management (koanf + validator)
│   │   └── server/               # HTTP server lifecycle & graceful shutdown
│   │
│   └── shared/                   # Cross-cutting primitives (no business logic)
│       ├── errors/               # Structured HTTP & domain errors
│       ├── response/             # Standard API envelopes
│       └── pagination/           # Pagination models
│
├── deployments/
│   ├── docker/                   # Multi-stage Dockerfiles
│   │   ├── Dockerfile.api
│   │   └── Dockerfile.worker
│   ├── nginx/                    # Nginx reverse proxy configuration
│   │   └── nginx.conf
│   └── docker-compose.yml        # Local development infrastructure
│
├── docs/
│   ├── architecture.md           # Architecture guidelines & dependency rules
│   └── database_models.md        # Database documentation & schemas
│
├── scripts/
│   └── setup.sh                  # Local bootstrap script
│
├── tests/
│   ├── integration/              # Integration test suites
│   └── load/                     # Load and performance tests
│
├── .env.example                  # Environment configuration blueprint
├── .gitignore                    # Git ignore file
├── Makefile                      # Standard development commands (Make)
├── Taskfile.yml                  # Task runner workflow pipeline (Task)
└── go.mod                        # Go module definition
```

---

## 2. Architectural Principles

1. **Feature-Based Modular Monolith**: Code is partitioned by business capabilities (`user`, `post`, `like`, `comment`, `follow`, `views`, `notification`) rather than horizontal technical layers.
2. **Clean Architecture Inside Modules**: Each module has its own `domain/`, `usecase/`, `repository/`, and `handler/http/` packages. Domain logic remains independent of databases and frameworks.
3. **Repository Interface Colocation**: Repository interfaces are defined inside each module's `repository/` package, close to the use cases that consume them.
4. **Platform Infrastructure Isolation**: Concrete drivers for MongoDB, PostgreSQL, Redis, and Kafka remain isolated inside `internal/platform/`.
5. **Dependency Inversion**: Dependencies flow inward. Use cases depend only on domain entities and repository interfaces. Implementations are injected through constructors.
6. **No Circular Dependencies**: Feature modules must not import each other cyclically.

For more details on architecture and data models, see:
- [docs/architecture.md](docs/architecture.md)
- [docs/database_models.md](docs/database_models.md)

---

## 3. System Architecture & Module Sequence Diagrams

`engage-engine` handles intense social engagement spikes using **in-memory probabilistic data structures**, **Kafka message streaming**, and **asynchronous micro-batch consumers** to eliminate row-level lock contention on PostgreSQL.

### 3.1. Overall Distributed Flow

```mermaid
sequenceDiagram
    autonumber
    actor Client as User / Browser
    participant API as API Server (Go HTTP Mux)
    participant Redis as Redis (Cache / HLL)
    participant Kafka as Kafka Broker
    participant Worker as Background Micro-Batch Worker
    participant DB as PostgreSQL

    Client->>API: HTTP Request (Like, View, Follow, Comment)
    alt Fast-Path Write (Views / Follows Surge)
        API->>Redis: Record Fast State (HLL PFADD / Realtime Counter)
        API-->>Kafka: Async Publish Event (engage.views / engage.follows)
        API-->>Client: 202 Accepted / 200 OK (Instant Response < 2ms)
        loop Micro-Batch Consumer (Every 200ms or 500 events)
            Worker->>Kafka: FetchBatch(topic)
            Worker->>DB: Batch Upsert / Atomic Increment (Single Round-Trip)
            Worker->>Kafka: CommitOffset()
        end
    else Direct ACID Write (Comments / Threaded Replies)
        API->>DB: Transactional INSERT + Counter Update
        API-->>Client: 201 Created / 200 OK
    end
```

---

### 3.2. Likes Module (`internal/modules/like`)

Coordinates liking and unliking of posts and comments with idempotent database updates and probabilistic deduplication via Cuckoo Filters.

```mermaid
sequenceDiagram
    autonumber
    actor User as Authenticated User
    participant Router as HTTP Router
    participant Handler as LikeHandler
    participant UC as LikeUseCase
    participant Filter as Cuckoo Filter
    participant Repo as LikeRepository
    participant DB as PostgreSQL (likes / posts)

    User->>Router: POST /api/v1/posts/{post_id}/likes
    Router->>Handler: LikePost(w, r)
    Handler->>UC: LikePost(ctx, userID, postID)
    UC->>Repo: LikePost(ctx, userID, postID)
    Repo->>DB: INSERT INTO likes (id, user_id, post_id, created_at)<br/>ON CONFLICT (user_id, post_id) DO NOTHING
    alt New Like Recorded
        Repo->>DB: UPDATE posts SET likes_count = likes_count + 1 WHERE id = $1
        Repo-->>UC: isNew=true, likesCount=N
        opt Cuckoo Filter Enabled
            UC->>Filter: RecordLike(userID, postID)
        end
        UC-->>Handler: LikeToggleOutput(liked=true, count=N)
        Handler-->>User: 200 OK {"liked": true, "likes_count": N}
    else Already Liked (Idempotent)
        Repo-->>UC: isNew=false, likesCount=N
        UC-->>Handler: LikeToggleOutput(liked=true, count=N)
        Handler-->>User: 200 OK {"liked": true, "likes_count": N}
    end
```

---

### 3.3. Comments Module (`internal/modules/comment`)

Supports threaded discussions (replies) and paginated comment feeds using keyset (cursor) pagination.

```mermaid
sequenceDiagram
    autonumber
    actor User as Authenticated User
    participant Router as HTTP Router
    participant Handler as CommentHandler
    participant UC as CommentUseCase
    participant Repo as CommentRepository
    participant DB as PostgreSQL (comments / users)

    User->>Router: POST /api/v1/posts/{post_id}/comments
    Note over User,Router: Body: {"content": "...", "parent_id": "uuid (optional)"}
    Router->>Handler: CreateComment(w, r)
    Handler->>UC: AddComment(ctx, CreateCommentInput)
    opt Threaded Reply (parent_id != nil)
        UC->>Repo: GetByID(ctx, parentID)
        Repo->>DB: SELECT * FROM comments WHERE id = $1
        DB-->>Repo: parentComment
        Repo-->>UC: parentComment
        Note over UC: Validate parent exists & belongs to same post_id
    end
    UC->>Repo: Create(ctx, comment)
    Repo->>DB: INSERT INTO comments (id, post_id, author_id, parent_id, content, ...)<br/>RETURNING id, created_at
    alt Top-Level Comment
        Repo->>DB: UPDATE posts SET comments_count = comments_count + 1 WHERE id = $1
    else Threaded Reply
        Repo->>DB: UPDATE comments SET replies_count = replies_count + 1 WHERE id = $parent_id
    end
    UC->>Repo: GetAuthor(ctx, authorID)
    Repo->>DB: SELECT id, username, display_name, avatar_url FROM users WHERE id = $1
    DB-->>Repo: Author Profile
    Repo-->>UC: Author Profile
    UC-->>Handler: CommentOutput(with enriched author details)
    Handler-->>User: 201 Created {"id": "...", "content": "...", "author": {...}}
```

---

### 3.4. Views & HyperLogLog Module (`internal/modules/views`)

High-throughput view tracking with **Redis HyperLogLog** for unique audience estimation (fixed 12KB memory) and **Kafka micro-batching** for persistent analytics in PostgreSQL.

```mermaid
sequenceDiagram
    autonumber
    actor Viewer as Client / Reader
    participant Router as HTTP Router
    participant Handler as ViewHandler
    participant UC as ViewUseCase
    participant Redis as Redis (HyperLogLog + Counter)
    participant Kafka as Kafka (engage.views)
    participant Worker as Views Micro-Batch Consumer
    participant DB as PostgreSQL (posts / post_daily_analytics)

    Viewer->>Router: POST /api/v1/views
    Note over Viewer,Router: Body: {"post_id": "uuid", "session_id": "fan_123", "duration_ms": 2500}
    Router->>Handler: RecordView(w, r)
    Handler->>UC: RecordView(ctx, in)
    UC->>Redis: PFADD analytics:hll:views:{post_id} {session_id}
    Note over Redis: 12KB fixed memory HyperLogLog<br/>O(1) cardinality deduplication
    UC->>Redis: INCR analytics:counter:views:{post_id}
    opt Kafka Producer Enabled
        UC--)Kafka: Async Publish ViewEvent (topic: engage.views)
    end
    UC-->>Handler: nil (Success)
    Handler-->>Viewer: 202 Accepted {"status": "recorded"}

    par Micro-Batch Worker Ingestion (Every 200ms or 500 events)
        Worker->>Kafka: FetchMessage()
        Note over Worker: Aggregates views per post into micro-batch map
        Worker->>DB: BEGIN TX
        Worker->>DB: UPDATE posts SET views_count = views_count + delta WHERE id = $post_id
        Worker->>DB: INSERT INTO post_daily_analytics (post_id, date, views) VALUES (...)<br/>ON CONFLICT (post_id, date) DO UPDATE SET views = views + delta
        Worker->>DB: COMMIT TX
        Worker->>Kafka: CommitMessages()
    end
```

---

### 3.5. Follows Module (`internal/modules/follow`)

Manages the social graph. Supports direct execution or high-throughput Kafka streaming with micro-batched database counter updates.

```mermaid
sequenceDiagram
    autonumber
    actor Follower as User A
    participant Router as HTTP Router
    participant Handler as FollowHandler
    participant UC as FollowUseCase
    participant Kafka as Kafka (engage.follows)
    participant Worker as Follows Micro-Batch Consumer
    participant DB as PostgreSQL (follows / users)

    Follower->>Router: POST /api/v1/users/{user_id}/follow
    Router->>Handler: FollowUser(w, r)
    Handler->>UC: Follow(ctx, followerID, followeeID)
    Note over UC: Validate followerID != followeeID
    alt Synchronous Direct Mode
        UC->>DB: INSERT INTO follows (follower_id, followee_id)<br/>ON CONFLICT DO NOTHING
        UC->>DB: UPDATE users SET following_count = following_count + 1 WHERE id = follower_id
        UC->>DB: UPDATE users SET followers_count = followers_count + 1 WHERE id = followee_id
    else High-Throughput Streaming Mode (Simulator / Kafka)
        UC--)Kafka: Publish FollowEvent (topic: engage.follows)
        loop Background Micro-Batch Aggregation
            Worker->>Kafka: FetchMessage() (Batch up to 500)
            Worker->>DB: UPDATE users SET followers_count = followers_count + delta<br/>WHERE id = followee_id
            Worker->>Kafka: CommitMessages()
        end
    end
    UC-->>Handler: FollowToggleOutput(is_following=true)
    Handler-->>Follower: 200 OK {"user_id": "...", "is_following": true}
```

---

### 3.6. Traffic Simulator & Consumer Drain (`cmd/simulate`)

Simulates high-velocity traffic (up to 1,000,000+ RPS) and actively monitors background consumer drain progress.

```mermaid
sequenceDiagram
    autonumber
    actor Admin as Simulator CLI
    participant Producer as cmd/simulate (30 Goroutines)
    participant Broker as Kafka Cluster (engage.follows)
    participant Consumer as Micro-Batch Worker
    participant DB as PostgreSQL
    participant Watcher as Consumer Drain Watcher

    Admin->>Producer: go run cmd/simulate/main.go -followers 1000000
    Producer->>DB: SELECT followers_count (Record initialCount)
    Note over Producer,Broker: Phase 1: High-Throughput Burst Dispatch
    loop 30 Concurrent Workers
        Producer->>Broker: PublishBatch (500 events per call)
    end
    Producer->>Admin: ✅ Dispatched 1,000,000 events to Kafka in 1.15s (864,000 RPS)
    
    par Async Worker Draining
        loop Micro-Batch Processing
            Consumer->>Broker: Fetch micro-batch of 500 msgs
            Consumer->>DB: UPDATE users SET followers_count = followers_count + 500
            Consumer->>Broker: Commit offset
        end
    and Active Consumer Drain Watcher (every 200ms)
        Watcher->>DB: SELECT followers_count FROM users WHERE id = target
        Watcher->>Admin: ⏳ [Consumer Drain] DB: 750,000 / 1,000,000 (75%) | Kafka Lag: ~250,000
        Watcher->>DB: SELECT followers_count FROM users WHERE id = target
        Watcher->>Admin: 🎯 [100% PERSISTED] All 1,000,000 followers committed to DB!
    end
```

---

## 4. Getting Started

### Prerequisites

- Go `1.22+`
- Make (or [Task](https://taskfile.dev))
- Docker & Docker Compose (optional for local containers)

### Setup

```bash
./scripts/setup.sh
```

### Common Commands

| Command (Make) | Command (Task) | Description |
| :--- | :--- | :--- |
| `make build` | `task build` | Compile `api` and `worker` binaries into `bin/` |
| `make run-api` | `task run-api` | Run the HTTP API server |
| `make run-worker` | `task run-worker` | Run the background worker service |
| `make test` | `task test` | Run unit tests with race detection |
| `make fmt` | `task fmt` | Format all Go source files |
| `make vet` | `task vet` | Run Go static analysis |
| `make clean` | `task clean` | Clean build artifacts |
| `make docker-up` | `task docker-up` | Spin up infrastructure containers |
| `make docker-down` | `task docker-down` | Tear down infrastructure containers |
