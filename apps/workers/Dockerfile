# syntax=docker/dockerfile:1

# Stage 1: build
FROM golang:1.24.4-alpine AS builder
WORKDIR /app

# Copy go.mod first for layer caching — workers has no go.sum (stdlib only)
COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/workers .

# Stage 2: minimal runtime image
FROM gcr.io/distroless/static:nonroot
COPY --from=builder /bin/workers /workers
EXPOSE 8081
ENTRYPOINT ["/workers"]
