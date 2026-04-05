# syntax=docker/dockerfile:1

# Stage 1: build
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Copy go.work and ALL module manifests (go work sync requires all modules)
COPY go.work go.work.sum* ./
COPY apps/workers/go.mod ./apps/workers/
COPY apps/graphql-api/go.mod apps/graphql-api/go.sum ./apps/graphql-api/
COPY packages/shared-go/go.mod packages/shared-go/go.sum* ./packages/shared-go/
RUN go work sync

# Copy source
COPY apps/workers/ ./apps/workers/
COPY packages/shared-go/ ./packages/shared-go/

RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/workers ./apps/workers/

# Stage 2: minimal runtime image
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /bin/workers /workers
EXPOSE 8081
ENTRYPOINT ["/workers"]
