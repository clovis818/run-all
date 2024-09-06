build: lint

fmt:
	go fmt ./...
	golines --dry-run ./
	golines -w ./

vet:
	go vet ./...
	staticcheck ./...

mod:
	go mod tidy
	go mod verify

sec:
	gosec -quiet -no-fail ./...
	govulncheck ./...
	trivy fs --include-dev-deps .

lint: mod fmt vet sec
	golangci-lint run

update:
	go get -u all

unit:
	go test --count=1 ./...

cov:
	go test --count=1 -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	go tool cover -html=coverage.out -o coverage.html
