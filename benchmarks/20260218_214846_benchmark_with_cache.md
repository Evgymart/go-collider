# Go-Collider Benchmark Report - with_cache

**Date:** Wed Feb 18 09:48:46 PM MSK 2026
**Configuration:**
- Host: http://localhost:8080
- Benchmark Name: with_cache
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
    Latency    82.34ms   65.06ms 535.47ms   74.34%
    Req/Sec   159.05     67.59   393.00     77.75%
  Latency Distribution
     50%   67.23ms
     75%  114.92ms
     90%  169.26ms
     99%  294.45ms
  6341 requests in 10.01s, 1.66MB read
Requests/sec:    633.57
Transfer/sec:    170.15KB
```


## Events List (GET /events)

```
Running 10s test @ http://localhost:8080/events?page=1&limit=100
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     1.31s   363.12ms   1.96s    62.39%
    Req/Sec     8.95      7.47    30.00     72.60%
  Latency Distribution
     50%    1.29s 
     75%    1.61s 
     90%    1.80s 
     99%    1.95s 
  219 requests in 10.01s, 3.40MB read
  Socket errors: connect 0, read 0, write 0, timeout 102
Requests/sec:     21.87
Transfer/sec:    347.61KB
```


## User Events (GET /users/{uid}/events)

```
Running 10s test @ http://localhost:8080
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    97.24ms   89.75ms 862.46ms   79.49%
    Req/Sec   143.25    115.81   530.00     85.53%
  Latency Distribution
     50%   93.30ms
     75%  112.30ms
     90%  191.48ms
     99%  481.26ms
  5539 requests in 10.02s, 34.89MB read
Requests/sec:    552.74
Transfer/sec:      3.48MB
```


## Stats (GET /stats)

```
Running 10s test @ http://localhost:8080
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    24.27ms   21.23ms 284.31ms   78.89%
    Req/Sec   564.72    182.81     1.02k    66.00%
  Latency Distribution
     50%   18.99ms
     75%   32.88ms
     90%   50.81ms
     99%   97.19ms
  22520 requests in 10.02s, 3.63MB read
Requests/sec:   2246.98
Transfer/sec:    370.82KB
```


---

**Benchmark completed at:** Wed Feb 18 09:49:26 PM MSK 2026

