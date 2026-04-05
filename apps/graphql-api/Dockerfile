# syntax=docker/dockerfile:1

# Stage 1: build
# gqlgen v0.17.89 requires go >= 1.25.0, so we use golang:1.25-alpine as the
# build image. The go.mod still declares go 1.24.4 as the minimum for the module
# itself, but the build toolchain must satisfy transitive dependencies.
FROM golang:1.25-alpine AS builder
WORKDIR /app

# Copy go.mod + go.sum first for layer caching
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/graphql-api .

# Stage 2: minimal runtime image
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /bin/graphql-api /graphql-api
EXPOSE 8080
ENTRYPOINT ["/graphql-api"]
