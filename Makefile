ARG=""

.PHONY: build

build:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-d -w -s' -o services/echo/main services/echo/cmd/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-d -w -s' -o services/private/main services/private/cmd/main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags '-d -w -s' -o services/task/main services/task/cmd/main.go

compose: build
	docker-compose up --build

up:
	docker-compose up

migrate-down:
	docker run -v `pwd`/migrations:/migrations --network envoy-sample_back migrate/migrate -path=/migrations/ -database "mysql://root:mysql@tcp(mysql:3306)/app?charset=utf8mb4" down