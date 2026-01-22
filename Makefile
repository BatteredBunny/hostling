build:
	CGO_ENABLED=0 go build -ldflags "-s -w" -o hostling ./cmd/hostling

clean:
	go clean
	rm -rf ./data ./hostling-data ./hostling.db

docker:
	docker build -t batteredbunny/hostling .

docker-push:
	docker push batteredbunny/hostling:latest