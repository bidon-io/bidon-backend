# CLAUDE.md

This file provides context and guidelines for working with the Bidon backend codebase.

## Project Overview

Bidon is a **Go-based ad mediation and programmatic advertising platform** that handles:
- Real-time bidding (RTB) auctions for mobile ads
- Ad unit configuration and management
- Demand source integration (adapters for various ad networks)
- User and app management through an admin API
- Event tracking and notifications (win/loss notifications)

**Key Technologies:**
- Language: Go
- Database: PostgreSQL (with GORM ORM)
- Caching: Redis
- Messaging: Redpanda (Kafka-compatible API)
- APIs: REST (Echo framework) + gRPC
- Protocol Buffers: For API definitions

## Project Structure

```
bidon-backend/
├── cmd/                    # Entry points for different services
│   ├── bidon-admin        # Admin API server
│   ├── bidon-sdkapi       # SDK API server (auction requests)
│   └── bidon-migrate      # Database migration tool
├── internal/              # Internal packages (core business logic)
│   ├── admin/            # Admin domain logic and API
│   ├── auction/          # Auction service (core bidding logic)
│   ├── bidding/          # Bidding adapters and OpenRTB
│   ├── notification/     # Win/loss notifications
│   ├── sdkapi/           # SDK-facing API logic
│   ├── adapter/          # Adapter abstractions
│   ├── segment/          # User segmentation
│   ├── ad/               # Ad type definitions
│   └── db/               # Database models (generated + hooks)
├── pkg/                   # Public/reusable packages
├── proto/                 # Protocol buffer definitions
├── config/                # Configuration files
├── docker/                # Docker-related files
└── scripts/               # Utility scripts
```

## Architecture Patterns

### 1. Repository Pattern
All data access is abstracted through repository interfaces:

**Structure:**
- `internal/*/store/*_repo.go` - Repository implementations
- Repositories wrap GORM operations
- Use generic `resourceRepo` for CRUD operations

**Example:** `internal/admin/store/line_item_repo.go`
```go
type LineItemRepo struct {
    *resourceRepo[admin.LineItem, admin.LineItemAttrs, db.LineItem]
}

func (r *LineItemRepo) Create(ctx context.Context, attrs *admin.LineItemAttrs) (*admin.LineItem, error)
func (r *LineItemRepo) Find(ctx context.Context, id int64) (*admin.LineItem, error)
func (r *LineItemRepo) List(ctx context.Context, qParams map[string][]string) (*resource.Collection[admin.LineItem], error)
```

**Common Patterns:**
- `List()` - Paginated list with filters
- `Find()` - Find by ID with associations
- `Create()` - Create with validation
- `Update()` - Update existing resource
- `Delete()` - Soft or hard delete
- `ListOwnedByUser()` - User-scoped queries (multi-tenancy)
- `FindOwnedByUser()` - User-scoped find

### 2. Service Layer
Business logic is in service structs:

**Structure:** `internal/*/service.go`

**Example:** `internal/auction/service.go`
```go
type Service struct {
    ConfigFetcher      ConfigFetcher
    AuctionBuilder     AuctionBuilder
    SegmentMatcher     *segment.Matcher
    AdapterKeysFetcher AdapterKeysFetcher
    EventLogger        *event.Logger
}

func (s *Service) Run(ctx context.Context, params *ExecutionParams) (*Response, error)
```

**Pattern:**
- Services depend on **interfaces** (for testability)
- Use dependency injection
- Handle orchestration and business rules

### 3. Handler/Controller Pattern
HTTP handlers in `internal/*/api/` or specific handler files:

**Example:** `internal/notification/handler.go`
```go
type Handler struct {
    AuctionResultRepo AuctionResultRepo
    Sender            Sender
    ConfigFetcher     ConfigFetcher
}

func (h Handler) HandleBiddingRound(ctx context.Context, ...) error
```

### 4. Domain Models
Three-layer model pattern:

1. **DB Models** (`internal/db/*.gen.go`) - Generated from database schema
2. **Domain Models** (`internal/admin/*.go`) - Business logic representation
3. **API Models** (`internal/admin/openapi/*.go`) - API request/response types

**Mappers** convert between layers (in repository files)

### 5. Adapters Pattern
Bidding adapters for different ad networks:

**Location:** `internal/bidding/adapters/`

Each adapter implements a common interface to interact with external demand sources.

### 6. Configuration Management
- Environment variables for runtime config
- Database-stored configs (auction configurations, line items)
- Config fetchers with caching (Redis)

## Common Code Templates

### Creating a New Repository

```go
package adminstore

import (
    "context"
    "gorm.io/gorm"
    "github.com/bidon-io/bidon-backend/internal/admin"
    "github.com/bidon-io/bidon-backend/internal/db"
)

type MyResourceRepo struct {
    *resourceRepo[admin.MyResource, admin.MyResourceAttrs, db.MyResource]
}

func NewMyResourceRepo(d *db.DB) *MyResourceRepo {
    return &MyResourceRepo{
        resourceRepo: &resourceRepo[admin.MyResource, admin.MyResourceAttrs, db.MyResource]{
            db:           d,
            mapper:       myResourceMapper{db: d},
            associations: []string{"RelatedEntity"},
        },
    }
}

type myResourceMapper struct {
    db *db.DB
}

func (m myResourceMapper) dbModel(attrs *admin.MyResourceAttrs, id int64) *db.MyResource {
    // Convert attrs to db model
}

func (m myResourceMapper) domainModel(model *db.MyResource) *admin.MyResource {
    // Convert db model to domain model
}
```

### Creating a New Service

```go
package mypackage

import "context"

type Service struct {
    MyRepo MyRepository
    // Other dependencies
}

//go:generate go run -mod=mod github.com/matryer/moq@v0.5.3 -out mocks/mocks.go -pkg mocks . MyRepository

type MyRepository interface {
    Find(ctx context.Context, id int64) (*MyResource, error)
}

func (s *Service) DoSomething(ctx context.Context, params *Params) (*Result, error) {
    // Implementation
}
```

### Adding a Handler Method

```go
type Handler struct {
    Service MyService
}

func (h *Handler) HandleRequest(ctx context.Context, req *Request) (*Response, error) {
    // Validation

    // Call service
    result, err := h.Service.DoSomething(ctx, params)
    if err != nil {
        return nil, err
    }

    // Transform to response
    return &Response{...}, nil
}
```

## Development Workflow

### Setup
```bash
# Initialize local environment
make local-init

# Full local stack (Postgres, Redis, Redpanda, APIs, UI)
docker compose -f docker-compose.dev.yml up -d

# Or start core dependencies only (Postgres, Redis)
docker compose up -d
```

### Running Services
```bash
# Admin API (port 1323)
go run ./cmd/bidon-admin

# SDK API (auction endpoint)
go run ./cmd/bidon-sdkapi
```

### Database Migrations
```bash
# Run migrations
go run ./cmd/bidon-migrate -help

# Apply migrations
go run ./cmd/bidon-migrate up
```

### Testing
```bash
# Run all tests
make test

# Run specific package tests
go test ./internal/auction/...
```

### Code Generation
```bash
# Generate mocks (run in package directory)
go generate ./...
```

## Key Concepts

### Advisory Locking
Used in repositories to prevent race conditions during concurrent operations:
```go
tx.Exec("SELECT pg_advisory_xact_lock(?)", lockKey)
```
Example: `LineItemRepo.firstOrCreate()` uses advisory locks for deduplication

### User Scoping (Multi-tenancy)
Resources are typically scoped to users:
- Use `ListOwnedByUser()` and `FindOwnedByUser()` methods
- Filter queries by `user_id`

### Auction Flow
1. SDK sends auction request to `/v2/auction`
2. Service matches segment and fetches auction config
3. Builds auction with demand sources (line items + bidding)
4. Runs bidding round (parallel adapter requests)
5. Returns ranked ad units to SDK
6. Handles win/loss notifications

### Event Logging
Events are logged to Redpanda for analytics:
- Auction events
- Impression events
- Click events

## Testing Conventions

- Test files: `*_test.go`
- Use `testify` for assertions
- Mock generation with `moq`
- Database tests use `dbtest` package
- Table-driven tests preferred

## Code Style

- Follow standard Go conventions
- Use `golangci-lint` for linting
- Interfaces defined in consuming packages
- Error handling: explicit, no panic in production code
- Context passed as first parameter

## Configuration

- `.env.sample` - Example environment variables
- `config/` - YAML configuration files
- Database config stored in `auction_configurations` table

## Resources

- [Self-Hosted Deployment Guide](https://docs.bidon.org/docs/server/self-hosted)
- OpenAPI specs: `internal/admin/openapi/`
- Proto definitions: `proto/`
