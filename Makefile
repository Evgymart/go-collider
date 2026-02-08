.PHONY: up down test clean-db

up:
	docker-compose up --build -d

down:
	docker-compose down

down-v:
	docker-compose down -v

test:
	docker-compose run --rm test
