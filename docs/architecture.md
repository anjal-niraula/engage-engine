# Architecture Guide: Feature-Based Modular Monolith

This document outlines the architectural standards and design patterns for `engage-engine`. The project is structured as a **Feature-Based Modular Monolith** where each functional domain is encapsulated into an independent module following **Clean Architecture** principles.

---

## 1. Core Architectural Principles

1. **Modular Monolith**: Code is organized around business features (`user`, `post`, `like`, `comment`, `follow`, `views`, `notification`) rather than technical layers. The entire codebase deploys as a unified binary, avoiding microservice operational overhead while preserving clean domain boundaries.
2. **Clean Architecture Inside Modules**: Each feature module internally enforces Clean Architecture:
   - `domain/`: Business entities and domain-specific rules (zero external dependencies).
   - `repository/`: Persistence interfaces (ports) declared close to the domain and use cases.
   - `usecase/`: Application business rules and workflows (interactors).
   - `handler/http/`: Transport delivery layer (HTTP routing, parameter binding, response encoding).
3. **Dependency Inversion (DIP)**: Use cases depend on repository interfaces, not concrete database drivers. Platform implementations satisfy repository interfaces.
4. **No Circular Dependencies**: Modules must not import each other cyclically. Inter-module communication is performed via clearly defined interfaces or asynchronous event streams (Kafka).
5. **Platform Isolation**: Database clients (MongoDB, PostgreSQL, Redis, Kafka) and cross-cutting platform concerns (Logger, Observability, Config) reside in `internal/platform/`.

---

## 2. Directory Layout

```text
engage-engine/
├── cmd/                          # Executable entry points (package main)
│   ├── api/                      # HTTP API service entry point
│   │   └── main.go
│   └── worker/                   # Asynchronous background worker service entry point
│       └── main.go
│
├── internal/
│   ├── modules/                  # Feature-based business modules
│   │   ├── user/                 # Profile and account management
│   │   │   ├── domain/           # User entity and value objects
│   │   │   ├── usecase/          # User business use cases
│   │   │   ├── repository/       # UserRepository interface
│   │   │   └── handler/          # HTTP route handlers
│   │   ├── post/                 # Post creation, media, feeds
│   │   │   ├── domain/
│   │   │   ├── usecase/
│   │   │   ├── repository/
│   │   │   └── handler/
│   │   ├── like/                 # Likes and engagement reactions
│   │   │   ├── domain/
│   │   │   ├── usecase/
│   │   │   ├── repository/
│   │   │   └── handler/
│   │   ├── comment/              # Comments and threaded discussions
│   │   │   ├── domain/
│   │   │   ├── usecase/
│   │   │   ├── repository/
│   │   │   └── handler/
│   │   ├── follow/               # Social graph and relationships
│   │   │   ├── domain/
│   │   │   ├── usecase/
│   │   │   ├── repository/
│   │   │   └── handler/
│   │   ├── notification/         # Alerts and activity notifications
│   │   │   ├── domain/
│   │   │   ├── usecase/
│   │   │   ├── repository/
│   │   │   └── handler/
│   │   └── views/                # Impression tracking, Redis HLL & view counters
│   │       ├── domain/
│   │       ├── usecase/
│   │       ├── repository/
│   │       └── handler/
│   │
│   ├── platform/                 # Infrastructure and technical adapters
│   │   ├── mongodb/              # Primary document database client & helpers
│   │   ├── postgres/             # Relational database client & pool
│   │   ├── redis/                # Cache, fast counters, HyperLogLog
│   │   ├── kafka/                # Event streaming producer/consumer clients
│   │   ├── logger/               # Structured logging abstractions (zerolog)
│   │   ├── config/               # Environment variable parsing and typed config (koanf)
│   │   └── server/               # HTTP server lifecycle & graceful shutdown
│   │
│   └── shared/                   # Cross-cutting primitives (no business logic)
│       ├── errors/               # Structured HTTP & domain errors
│       ├── response/             # Standard API JSON envelopes
│       └── pagination/           # Cursor and offset pagination types
│
├── deployments/                  # Containerization and networking
│   ├── docker/                   # Multi-stage Dockerfiles for API and Worker
│   │   ├── Dockerfile.api
│   │   └── Dockerfile.worker
│   ├── nginx/                    # Reverse proxy and load balancing config
│   │   └── nginx.conf
│   └── docker-compose.yml        # Local infrastructure stack
│
├── docs/                         # Project documentation
│   ├── architecture.md           # Architecture overview (this file)
│   └── database_models.md        # Schemas and database models
│
├── scripts/                      # Automation scripts
│   └── setup.sh                  # Local development bootstrap
│
├── tests/                        # Testing suites
│   ├── integration/              # Live dependency integration tests
│   └── load/                     # Performance and stress tests
│
├── .env.example                  # Environment configuration blueprint
├── .gitignore                    # Git ignore file
├── Makefile                      # Standard development commands (GNU Make)
├── Taskfile.yml                  # Task runner workflow pipeline (Task)
└── go.mod                        # Go module definition
```

---

## 3. Dependency Flow & Architecture Rules

```text
[ HTTP Client / Nginx ]
          │
          ▼
[ cmd/api/main.go ] ──── (wires dependencies via constructors)
          │
          ▼
[ handler/http/ ] ────── (extracts DTO, validates parameters)
          │
          ▼
[ usecase/ ] ─────────── (executes business rules)
     │         │
     ▼         ▼
[ domain/ ]   [ repository/ ] (interface port)
                     ▲
                     │ (implements port)
              [ platform/ ] (MongoDB, PostgreSQL, Redis, Kafka)
```

1. **Dependency Inward Rule**: High-level modules do not import low-level infrastructure drivers directly.
2. **Module Independence**: Modules communicate with each other through explicit service interfaces or Kafka events rather than reaching into another module's internal repository.
3. **Constructor Injection**: All dependencies (repositories into use cases, use cases into handlers) are injected explicitly via constructors (`New(...)`).
