# Go-Collider

A highload practice sandbox for event tracking and analytics.

## What is this?

Go-collider is a lightweight event tracking API built with Go and PostgreSQL. It's designed as a playground for practicing performance optimization techniques on large datasets (10M+ events).

## Quick Start

```bash
make up       # Start Docker containers
make seed     # Seed database with test data
make run      # Run the API
```

Run `make help` to see all available commands.

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/events` | List events (paginated) |
| POST | `/events` | Create event |
| GET | `/users/{id}/events` | Get user's events |
| GET | `/stats` | Get analytics |

## Benchmarking

```bash
make bench                  # Run benchmark
make bench NAME=naive       # Allows name as a param
```

Results are saved to `benchmarks/benchmark_{name}_{datetime}.md`.
