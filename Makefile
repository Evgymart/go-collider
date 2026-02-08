.PHONY: up down test

up:
	docker-compose up --build -d

down:
	docker-compose down -v

test:
	docker-compose run --rm test
