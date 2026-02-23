g# Go-Collider Benchmark Report - optimized_retrieve

**Date:** Mon Feb 23 11:59:04 PM MSK 2026
**Configuration:**
- Host: http://localhost:8080
- Benchmark Name: optimized_retrieve
- Threads: 4
- Connections: 50
- Duration: 10s per test
- User ID: 1
- Event Type: page_view

---


## Create Event (POST /events)

```
Running 10s test @ http://localhost:8080/events
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    25.94ms   27.86ms 103.92ms   79.28%
    Req/Sec   702.40    554.13     3.99k    83.29%
  Latency Distribution
     50%   11.03ms
     75%   48.62ms
     90%   71.00ms
     99%   89.76ms
  28019 requests in 10.10s, 8.15MB read
Requests/sec:   2774.26
Transfer/sec:    826.02KB
```


## Events List (GET /events)

```
Running 10s test @ http://localhost:8080/events?page=1&limit=100
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    49.49ms   44.70ms 376.24ms   82.48%
    Req/Sec   257.09    129.45   565.00     60.61%
  Latency Distribution
     50%   25.44ms
     75%   88.56ms
     90%   97.10ms
     99%  178.73ms
  10190 requests in 10.01s, 167.91MB read
Requests/sec:   1017.98
Transfer/sec:     16.77MB
```


## User Events (GET /users/{uid}/events)

```
Running 10s test @ http://localhost:8080
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    33.71ms   39.02ms 444.58ms   82.62%
    Req/Sec   498.01    230.29     1.08k    66.58%
  Latency Distribution
     50%   14.81ms
     75%   58.34ms
     90%   83.21ms
     99%  166.66ms
  19456 requests in 10.01s, 130.08MB read
Requests/sec:   1944.47
Transfer/sec:     13.00MB
```


## Stats (GET /stats)

```
Running 10s test @ http://localhost:8080
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    25.65ms   35.85ms 429.62ms   84.83%
    Req/Sec     0.92k   684.80     3.55k    84.05%
  Latency Distribution
     50%    5.93ms
     75%   44.52ms
     90%   68.79ms
     99%  151.39ms
  36207 requests in 10.01s, 5.87MB read
Requests/sec:   3618.10
Transfer/sec:    600.66KB
```


---

**Benchmark completed at:** Mon Feb 23 11:59:44 PM MSK 2026

