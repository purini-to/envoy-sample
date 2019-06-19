.PHONY: build

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-d -w -s' -o services/echo/main services/echo/cmd/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-d -w -s' -o services/private/main services/private/cmd/main.go

compose: build
	docker-compose up --build

up:
	docker-compose up