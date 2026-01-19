FROM node:23-alpine AS frontend

WORKDIR /app/cmd/frontend

RUN corepack enable && corepack prepare pnpm@latest --activate

COPY cmd/frontend/package.json cmd/frontend/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY cmd/frontend/ ./
RUN pnpm run build

FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

COPY --from=frontend /app/cmd/public/dist ./cmd/public/dist

RUN go build -o /app/hostling

FROM alpine:3.23

VOLUME [ "/app/data" ]
EXPOSE 80
WORKDIR /app

COPY --from=builder /app/hostling /app/hostling

ENTRYPOINT [ "/app/hostling", "-c", "/app/config.toml" ]