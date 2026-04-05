# Project Neo - Development Roadmap

**Last Updated**: April 5, 2026 (session 2)

> Progress tracking lives here. Session context (commands, conventions, architecture) is in CLAUDE.md.

## Phase 1: Development Infrastructure Setup

### ✅ Completed
- [x] Initialize Git repository
- [x] Configure Git user (rahulmathews)
- [x] Add remote origin (GitHub)
- [x] Install Bun v1.3.11
- [x] Initialize Turborepo monorepo
- [x] Set Node.js 24 LTS requirements (.nvmrc)
- [x] Add Biome for formatting/linting (10-25x faster than Prettier+ESLint)
- [x] Configure Husky for git hooks
- [x] Configure lint-staged for pre-commit checks
- [x] Add commitlint for conventional commits enforcement
- [x] Create commit-msg hook
- [x] Add Changesets for version management
- [x] Create GitHub Actions CI/CD workflows
  - [x] CI workflow (lint, format check, build)
  - [x] Release workflow (changesets, GitHub releases)

---

## Phase 2: Containerization & Docker Setup

### ✅ Completed
- [x] Create root docker-compose.yml for local development
  - [x] Workers service
  - [x] GraphQL API service
- [x] Create .env.example
- [x] Create individual Dockerfiles for each service
  - [x] `apps/workers/Dockerfile`
  - [x] `apps/graphql-api/Dockerfile`
- [x] Add Docker-related scripts to Makefile
  - [x] `docker-up` - Start all services
  - [x] `docker-down` - Stop all services
  - [x] `docker-build` - Build all images
  - [x] `docker-logs` - View logs
  - [x] `dev-up` / `dev-down` - Start/stop everything
- [x] Add health checks to containers
- [x] Set up environment variable templates (.env.example)

### 📋 Production Docker Setup (Pending)
- [x] Multi-stage builds (golang:1.25-alpine → distroless)
- [x] Non-root user (distroless:nonroot)
- [x] Minimal base image (distroless/static)
- [ ] Security scanning
- [ ] Create docker-compose.prod.yml
- [ ] Add Kubernetes manifests (optional, future)

---

## Phase 3: Core Application Structure

### ✅ Completed — Supabase Setup
- [x] Install Supabase CLI
- [x] Initialize Supabase locally (`supabase init`)
- [x] Design database schema (8 tables — models exist in shared-go)

### ✅ Completed — Supabase Migrations
- [x] Write SQL migration files
  - [x] `20260331000000_baseline.sql` — all 8 tables (users, groups, group_sources, location_contexts, messages, rides, matches, locations)
  - [x] `20260404000000_notify_triggers.sql` — NOTIFY triggers for GraphQL subscriptions
- [x] Set up Row Level Security (RLS) policies
- [x] Add updated_at triggers
- [x] Add NOTIFY triggers for GraphQL subscriptions (`rides_added`, `rides_updated`, `matches_updated`)
- [x] Add indexes for query performance

### 📋 Supabase Pending
- [ ] Configure Auth providers
- [ ] Add database seed data for development

### ✅ Completed — Shared Go Package (`packages/shared-go`)
- [x] Create `packages/shared-go` module (`project-neo/shared`)
- [x] Domain models (`model/`): User, Group, Location, LocationContext, Ride, Match, inputs
- [x] Repository interfaces (`repository/`): User, Group, Ride, Match, Location
- [x] PostgreSQL implementations (`postgres/`): all entities via Bun ORM
- [x] Add to go.work workspace
- [x] Update all import paths in graphql-api

### ✅ Completed — GraphQL API (`apps/graphql-api`)
- [x] Initialize Go module
- [x] Initialize gqlgen with schema definitions
- [x] Define GraphQL schema (types, queries, mutations, subscriptions)
- [x] Implement internal domain models
- [x] Implement postgres repository layer (all entities)
- [x] Implement pg_notify listener + broker for subscriptions
- [x] Implement query resolvers
- [x] Implement mutation resolvers
- [x] Implement subscription resolvers
- [x] Wire auth middleware
- [x] Health endpoint verified

### ✅ Completed — Workers Service (`apps/workers`)
- [x] Initialize Go module
- [x] Set up worker framework
- [x] Health endpoint verified

### 📋 Workers Service — Pending
- [ ] Set up message source connectors
  - [ ] WhatsApp connector (whatsmeow)
  - [ ] Telegram connector (go-telegram-bot-api)
  - [ ] Manual entry connector
- [ ] Implement message parser
  - [ ] NLP/regex-based extraction
  - [ ] Location context resolution
  - [ ] Time parsing ("now", "3:30 PM", etc.)
- [ ] Background job processing
  - [ ] Message polling/listening
  - [ ] Parse and store messages
  - [ ] Error handling and retries
- [ ] Add logging and monitoring

### 📋 Flutter Mobile App (`apps/mobile`) — Not Started
- [ ] Initialize Flutter project
- [ ] Set up project structure
  - [ ] Features-based architecture
  - [ ] State management (Riverpod/Bloc/Provider - TBD)
- [ ] Core features
  - [ ] Authentication (Supabase Auth)
  - [ ] Ride listing (available rides)
  - [ ] Ride requests (need ride)
  - [ ] User matching UI
  - [ ] Real-time updates (GraphQL subscriptions)
  - [ ] Location context configuration
- [ ] UI/UX
  - [ ] Material Design + Cupertino widgets
  - [ ] Dark mode support
  - [ ] Responsive layouts
- [ ] Platform-specific setup
  - [ ] iOS configuration
  - [ ] Android configuration
  - [ ] Permissions (location, notifications)

---

## Phase 4: Monitoring & Observability

### 📋 Pending
- [ ] Add structured logging to Go services
- [ ] Set up error tracking (Sentry or self-hosted alternative)
- [ ] Add metrics collection (Prometheus/Grafana - optional)
- [ ] Database query performance monitoring
- [ ] API response time tracking

---

## Phase 5: Documentation & Developer Experience

### 📋 Pending
- [ ] Create CONTRIBUTING.md
- [ ] Update README.md
  - [ ] Project overview
  - [ ] Quick start guide
  - [ ] Architecture diagram
  - [ ] Deployment instructions
- [ ] API documentation (GraphQL schema docs)
- [ ] Architecture Decision Records (ADRs)

---

## Phase 6: Deployment & DevOps

### 📋 Pending
- [ ] Create deployment scripts
- [ ] Set up reverse proxy (Nginx/Caddy)
- [ ] SSL/TLS certificates (Let's Encrypt)
- [ ] Environment configuration management
- [ ] Database backups automation
- [ ] Log rotation and management
- [ ] Automated deployment on merge to main
- [ ] Staging environment setup
- [ ] Database migration automation
- [ ] Rollback procedures

---

## Phase 7: Future Enhancements

### 📋 Advanced Features
- [ ] In-app messaging between riders and drivers
- [ ] Payment integration
- [ ] Route optimization algorithms
- [ ] Push notifications
- [ ] Analytics dashboard
- [ ] Admin panel
- [ ] Multi-language support
- [ ] Ratings and reviews system

### 📋 Performance Optimizations
- [ ] Caching layer (Redis)
- [ ] CDN for static assets
- [ ] Database query optimization
- [ ] GraphQL query optimization
- [ ] Mobile app performance profiling

### 📋 Security Enhancements
- [ ] Security audit
- [ ] Penetration testing
- [ ] Rate limiting
- [ ] Input validation hardening
- [ ] Secrets management (Vault/SOPS)
- [ ] GDPR compliance considerations

---

## Current Status Summary

**Next Immediate Tasks**:
1. Implement message source connectors + parser in workers service
2. Begin Flutter mobile app scaffold (`apps/mobile`)
