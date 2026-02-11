##
## Docker
## -------

up: ## Start Docker containers
	docker-compose up --build -d
.PHONY: up

down: ## Stop Docker containers
	docker-compose down
.PHONY: down

down-v: ## Stop Docker containers and remove volumes
	docker-compose down -v
.PHONY: down-v

##
## Development
## -----------

run: ## Run the API locally
	go run ./cmd/api/main.go
.PHONY: run

build: ## Build the binary
	go build -o collider ./cmd/api/main.go
.PHONY: build

test: ## Run tests in Docker
	docker-compose run --rm test go test ./test/cases/...
.PHONY: test

test-local: ## Run tests locally
	go test ./test/cases/...
.PHONY: test-local

bench: ## Run benchmarks (usage: make bench NAME=optimized)
	./scripts/benchmark.sh http://localhost:8080 $(NAME)
.PHONY: bench

##
## Database
## --------

seed: ## Seed database with test data
	docker-compose run --rm test sh -c 'go run cmd/seed/main.go'
.PHONY: seed

seed-prod: ## Seed database with production-like data (10M events)
	docker-compose run --rm test sh -c 'go run cmd/seed/main.go --prod'
.PHONY: seed-prod

truncate: ## Truncate all database tables
	docker-compose run --rm test sh -c 'go run cmd/truncate/main.go'
.PHONY: truncate

# -----------------------

help:
	@awk ' \
		BEGIN {RS=""; FS="\n"} \
		function printCommand(line) { \
			split(line, command, ":.*?## "); \
			printf "\033[32m%-20s\033[0m %s\n", command[1], command[2]; \
		} \
		/^[0-9a-zA-Z_-]+: [0-9a-zA-Z_-]+\n[0-9a-zA-Z_-]+: .*?##.*$$/ { \
			split($$1, alias, ": "); \
			sub(alias[2] ":", alias[2] " (" alias[1] "):", $$2); \
			printCommand($$2); \
			next; \
		} \
		$$1 ~ /^[0-9a-zA-Z_-]+: .*?##/ { \
			printCommand($$1); \
			next; \
		} \
		/^##(\n##.*)+$$/ { \
			gsub("## ?", "\033[33m", $$0); \
			print $$0; \
			next; \
		} \
	' $(MAKEFILE_LIST) && printf "\033[0m"
.PHONY: help

.DEFAULT_GOAL := help
