build:
    go run ./cmd/openapi frontend/openapi.json
    pnpm --dir frontend build
    CGO_ENABLED=0 go build -ldflags "-s -w" -o hostling ./cmd/hostling

migration:
    atlas migrate diff --env sqlite
    atlas migrate diff --env postgresql

clean:
    go clean
    rm -rf ./data ./hostling-data ./hostling.db

docker:
    docker build -t batteredbunny/hostling:latest .

docker-push:
    docker push batteredbunny/hostling:latest

lint:
    golangci-lint run ./...
    pnpm --dir frontend lint

fmt:
    golangci-lint fmt ./...
    pnpm --dir frontend lint:fix
    pnpm --dir frontend format

dev *args='-c examples/example_local_sqlite.toml':
    air -- {{args}}

generate-api:
    go run ./cmd/openapi frontend/openapi.json
    pnpm --dir frontend generate:api

check-api: generate-api
    git diff --exit-code -- frontend/openapi.json frontend/src/generated
