.PHONY: build clean run test lint docker

BINARY=prorouter
GO_DIR=gateway-go

build:
	cd $(GO_DIR) && CGO_ENABLED=0 go build -o ../$(BINARY) ./cmd/prorouter

run: build
	./$(BINARY) serve

clean:
	rm -f $(BINARY)
	rm -rf $(GO_DIR)/prorouter
	rm -rf $(GO_DIR)/prorouter.exe

test:
	cd $(GO_DIR) && CGO_ENABLED=0 go vet ./...
	cd $(GO_DIR) && CGO_ENABLED=0 go test ./...

lint:
	cd $(GO_DIR) && gofmt -l -e .

docker:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

install:
	cp $(BINARY) /usr/local/bin/$(BINARY)

dev: build
	./$(BINARY) serve --port 8080

init:
	./$(BINARY) init

doctor:
	./$(BINARY) doctor

key-gen:
	./$(BINARY) key generate

e2e:
	npx playwright test

e2e-install:
	npx playwright install chromium
