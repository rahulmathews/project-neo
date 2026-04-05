# Project Neo - Development Roadmap

**Last Updated**: April 5, 2026

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
- [ ] Optimize Docker images for production (multi-stage builds)
- [ ] Add Docker security best practices
  - [ ] Non-root user
  - [ ] Minimal base images (Alpine/Distroless)
  - [ ] Security scanning
- [ ] Create docker-compose.prod.yml
- [ ] Add Kubernetes manifests (optional, future)

---

## Phase 3: Core Application Structure

### ✅ Completed — Supabase Setup
- [x] Install Supabase CLI
- [x] Initialize Supabase locally (`supabase init`)
- [x] Design and apply database schema migrations (8 tables)
  - [x] users table
  - [x] groups table
  - [x] group_sources table
  - [x] location_contexts table
  - [x] messages table
  - [x] rides table
  - [x] matches table
  - [x] locations table
- [x] Set up Row Level Security (RLS) policies
- [x] Add updated_at triggers
- [x] Add NOTIFY triggers for GraphQL subscriptions (pg_notify)
- [x] Add indexes for query performance

### 📋 Supabase Pending
- [ ] Configure Auth providers
- [ ] Add database seed data for development

### ✅ Completed — Shared Go Package (`packages/shared-go`)
- [x] Create `packages/shared-go` module (`project-neo/shared`)
- [x] Move domain models (`model/`) from graphql-api internal
- [x] Move repository interfaces (`repository/`) from graphql-api internal
- [x] Move postgres CRUD implementations (`postgres/`) from graphql-api internal
- [x] Add to go.work workspace
- [x] Update all import paths in graphql-api (resolvers, main, gqlgen.yml, generated)

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
1. Begin Flutter mobile app scaffold (`apps/mobile`)
2. Add database seed data for development
3. Implement message source connectors in workers service
