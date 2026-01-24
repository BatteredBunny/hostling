build:
	CGO_ENABLED=0 go build -ldflags "-s -w" -o hostling ./cmd/hostling

migration:
	atlas migrate diff --env sqlite
	atlas migrate diff --env postgresql

clean:
	go clean
	rm -rf ./data ./hostling-data ./hostling.db

docker:
	docker build -t batteredbunny/hostling .

docker-push:
	docker push batteredbunny/hostling:latest