.PHONY: test build check
build:
	go build -trimpath -o urbitctl ./cmd/urbitctl
test:
	go test -race ./...
check: test
	go vet ./...
	bash -n scripts/install.sh scripts/preflight.sh scripts/pilot.sh
