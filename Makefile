.PHONY: up down test clean-db seed truncate

up:
	docker-compose up --build -d

down:
	docker-compose down

down-v:
	docker-compose down -v

test:
	docker-compose run --rm test

seed:
	docker-compose run --rm test sh -c 'go run cmd/seed/main.go'

seed-prod:
	docker-compose run --rm test sh -c 'go run cmd/seed/main.go --prod'

truncate:
	docker-compose run --rm test sh -c 'go run cmd/truncate/main.go'
