# Go-Collider Benchmark Report - optimized_insert

**Date:** Wed Feb 18 11:01:29 PM MSK 2026
**Configuration:**
- Host: http://localhost:8080
- Benchmark Name: optimized_insert
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
    Latency    24.95ms   28.28ms 275.73ms   80.23%
    Req/Sec   787.85    450.38     2.48k    83.50%
  Latency Distribution
     50%    7.59ms
     75%   46.46ms
     90%   68.77ms
     99%   88.43ms
  31390 requests in 10.01s, 3.35MB read
Requests/sec:   3136.10
Transfer/sec:    343.01KB
```


## Events List (GET /events)

```
Running 10s test @ http://localhost:8080/events?page=1&limit=100
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     1.26s   364.16ms   1.93s    60.45%
    Req/Sec    11.93      8.00    40.00     64.04%
  Latency Distribution
     50%    1.34s 
     75%    1.52s 
     90%    1.68s 
     99%    1.90s 
  357 requests in 10.01s, 37.30KB read
  Socket errors: connect 0, read 0, write 0, timeout 3
Requests/sec:     35.66
Transfer/sec:      3.73KB
```


## User Events (GET /users/{uid}/events)

```
Running 10s test @ http://localhost:8080
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    30.26ms   14.54ms 174.81ms   74.38%
    Req/Sec   404.58     85.42   545.00     76.75%
  Latency Distribution
     50%   27.98ms
     75%   37.27ms
     90%   48.68ms
     99%   80.22ms
  16118 requests in 10.02s, 1.64MB read
Requests/sec:   1608.59
Transfer/sec:    168.09KB
```


## Stats (GET /stats)

```
Running 10s test @ http://localhost:8080
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    76.80ms   65.90ms 268.22ms   52.31%
    Req/Sec   166.40     35.14   262.00     67.25%
  Latency Distribution
     50%   81.54ms
     75%  131.69ms
     90%  171.00ms
     99%  203.29ms
  6640 requests in 10.02s, 693.83KB read
Requests/sec:    662.48
Transfer/sec:     69.22KB
```


---

**Benchmark completed at:** Wed Feb 18 11:02:09 PM MSK 2026

