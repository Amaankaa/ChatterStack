# syntax=docker/dockerfile:1
FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# No explicit GOARCH: build natively for the host (works on both amd64 and arm64 servers).
RUN CGO_ENABLED=0 go build -o chatterstack ./cmd/server

FROM gcr.io/distroless/base-debian12

WORKDIR /app

COPY --from=builder /app/chatterstack ./chatterstack

EXPOSE 8080

ENTRYPOINT ["./chatterstack"]