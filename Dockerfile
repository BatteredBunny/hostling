FROM node:25-alpine3.23 AS frontend

WORKDIR /app/frontend

RUN npm install -g pnpm@10

COPY frontend/package.json frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY frontend/ ./
RUN pnpm run build

FROM golang:1.26-alpine3.23 AS builder
RUN apk add --no-cache gcc musl-dev
WORKDIR /app
COPY go.mod go.sum ./

RUN go mod download

COPY . .

COPY --from=frontend /app/public/dist ./public/dist
RUN go build -ldflags '-s -w -extldflags "-static"' -o /app/hostling ./cmd/hostling

FROM alpine:3.23

RUN apk add --no-cache ca-certificates

VOLUME [ "/app/data" ]
EXPOSE 80
WORKDIR /app

COPY --from=builder /app/hostling /app/hostling

ENTRYPOINT [ "/app/hostling", "-c", "/app/config.toml" ]