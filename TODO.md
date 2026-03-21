# Project Neo - Development Roadmap

**Last Updated**: March 21, 2026

## Phase 1: Development Infrastructure Setup (Current)

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

### 🔄 In Progress
- [ ] Add commitlint for conventional commits enforcement
- [ ] Create commit-msg hook

### 📋 Pending (Current Phase)
- [ ] Add Changesets for version management
- [ ] Create GitHub Actions CI/CD workflows
  - [ ] CI workflow (lint, format check, build)
  - [ ] Release workflow (changesets, GitHub releases)
- [ ] Update CLAUDE.md with final tooling documentation

---

## Phase 2: Containerization & Docker Setup

### 📋 Docker Infrastructure
- [ ] Create root Dockerfile (multi-stage build)
- [ ] Create docker-compose.yml for local development
  - [ ] Supabase services (PostgreSQL, PostgREST, Realtime, Auth)
  - [ ] Workers service
  - [ ] GraphQL API service
  - [ ] PgAdmin or database management UI
- [ ] Create .dockerignore
- [ ] Create individual Dockerfiles for each service
  - [ ] `apps/workers/Dockerfile`
  - [ ] `packages/graphql-api/Dockerfile`
- [ ] Add Docker-related scripts to package.json
  - [ ] `docker:up` - Start all services
  - [ ] `docker:down` - Stop all services
  - [ ] `docker:build` - Build all images
  - [ ] `docker:logs` - View logs
- [ ] Document Docker setup in CLAUDE.md
- [ ] Add health checks to containers
- [ ] Configure volume mounts for development hot-reload
- [ ] Set up environment variable templates (.env.example)

### 📋 Production Docker Setup
- [ ] Optimize Docker images for production (multi-stage builds)
- [ ] Add Docker security best practices
  - [ ] Non-root user
  - [ ] Minimal base images (Alpine/Distroless)
  - [ ] Security scanning
- [ ] Create docker-compose.prod.yml
- [ ] Add Kubernetes manifests (optional, future)

---

## Phase 3: Core Application Structure

### 📋 Supabase Setup
- [ ] Initialize Supabase locally (`supabase init`)
- [ ] Design database schema (from CLAUDE.md)
  - [ ] users table
  - [ ] groups table
  - [ ] group_sources table
  - [ ] location_contexts table
  - [ ] messages table
  - [ ] rides table
  - [ ] matches table
  - [ ] locations table
- [ ] Create migration files
- [ ] Set up Row Level Security (RLS) policies
- [ ] Configure Auth providers
- [ ] Set up Realtime subscriptions
- [ ] Add database seed data for development

### 📋 Shared Packages
- [ ] Create `packages/shared-types` (TypeScript types)
- [ ] Create `packages/database` (Supabase client, schemas)
- [ ] Create `packages/graphql-api` (GraphQL server)
  - [ ] Schema definitions
  - [ ] Resolvers
  - [ ] Subscriptions for realtime
  - [ ] Authentication middleware
- [ ] Add proper TypeScript configuration

### 📋 Workers Service (apps/workers)
- [ ] Initialize Node.js/Bun project
- [ ] Set up message source connectors
  - [ ] WhatsApp connector (whatsapp-web.js/baileys)
  - [ ] Telegram connector (Telegram Bot API)
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
- [ ] Write unit tests

### 📋 Flutter Mobile App (apps/mobile)
- [ ] Initialize Flutter project
- [ ] Set up project structure
  - [ ] Features-based architecture
  - [ ] State management (Riverpod/Bloc/Provider - TBD)
- [ ] Core features
  - [ ] Authentication (Supabase Auth)
  - [ ] Ride listing (available rides)
  - [ ] Ride requests (need ride)
  - [ ] User matching UI
  - [ ] Real-time updates (subscriptions)
  - [ ] Location context configuration
- [ ] UI/UX
  - [ ] Material Design + Cupertino widgets
  - [ ] Dark mode support
  - [ ] Responsive layouts
- [ ] Platform-specific setup
  - [ ] iOS configuration
  - [ ] Android configuration
  - [ ] Permissions (location, notifications)
- [ ] Testing
  - [ ] Unit tests
  - [ ] Widget tests
  - [ ] Integration tests

---

## Phase 4: Testing & Quality Assurance

### 📋 Testing Infrastructure
- [ ] Set up Vitest/Bun test runner for backend
- [ ] Configure test coverage reporting
- [ ] Add GitHub Actions for running tests
- [ ] Integration tests for GraphQL API
- [ ] E2E tests for critical user flows
- [ ] Flutter widget and integration tests

### 📋 Monitoring & Observability
- [ ] Add structured logging (Winston/Pino)
- [ ] Set up error tracking (Sentry or self-hosted alternative)
- [ ] Add metrics collection (Prometheus/Grafana - optional)
- [ ] Database query performance monitoring
- [ ] API response time tracking

---

## Phase 5: Documentation & Developer Experience

### 📋 Documentation
- [ ] Complete CLAUDE.md with all setup instructions
- [ ] Create CONTRIBUTING.md
  - [ ] Development workflow
  - [ ] Commit conventions
  - [ ] PR process
  - [ ] Changeset creation guide
- [ ] Update README.md
  - [ ] Project overview
  - [ ] Quick start guide
  - [ ] Architecture diagram
  - [ ] Deployment instructions
- [ ] API documentation (GraphQL schema docs)
- [ ] Database schema documentation
- [ ] Architecture Decision Records (ADRs)

### 📋 Developer Tools
- [ ] VS Code workspace settings
- [ ] Recommended VS Code extensions list
- [ ] Debug configurations for VS Code
- [ ] Cursor/Copilot rules (if applicable)

---

## Phase 6: Deployment & DevOps

### 📋 VPS Deployment
- [ ] Create deployment scripts
- [ ] Set up reverse proxy (Nginx/Caddy)
- [ ] SSL/TLS certificates (Let's Encrypt)
- [ ] Environment configuration management
- [ ] Database backups automation
- [ ] Log rotation and management
- [ ] Monitoring and alerts

### 📋 CI/CD Enhancement
- [ ] Automated deployment on merge to main
- [ ] Staging environment setup
- [ ] Database migration automation
- [ ] Rollback procedures
- [ ] Performance testing in CI

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

**Completed**: 9 tasks
**In Progress**: 2 tasks
**Pending in Current Phase**: 4 tasks
**Total Roadmap Items**: 100+

**Next Immediate Tasks**:
1. Complete commitlint setup
2. 
2. Add Changesets
3. Create GitHub Actions workflows
4. Begin Docker containerization (Phase 2)
