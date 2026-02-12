# Go-Collider Benchmark Report - indexes

**Date:** Thu Feb 12 10:15:40 PM MSK 2026
**Configuration:**
- Host: http://localhost:8080
- Benchmark Name: indexes
- Threads: 4
- Connections: 50
- Duration: 10s per test
- User ID: bc1e30d0-221f-4440-bfe7-0fb3f2fd43bb
- Event Type: page_view

---


## Create Event (POST /events)

```
Running 10s test @ http://localhost:8080/events
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency   101.38ms   74.10ms 584.87ms   71.32%
    Req/Sec   126.03     32.12   282.00     69.75%
  Latency Distribution
     50%   87.01ms
     75%  138.97ms
     90%  199.32ms
     99%  341.42ms
  5026 requests in 10.01s, 1.81MB read
Requests/sec:    502.08
Transfer/sec:    184.85KB
```


## Events List (GET /events)

```
Running 10s test @ http://localhost:8080/events?page=1&limit=100
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     0.00us    0.00us   0.00us    -nan%
    Req/Sec     5.08      6.72    30.00     91.89%
  Latency Distribution
     50%    0.00us
     75%    0.00us
     90%    0.00us
     99%    0.00us
  48 requests in 10.01s, 1.44MB read
  Socket errors: connect 0, read 0, write 0, timeout 48
Requests/sec:      4.79
Transfer/sec:    147.69KB
```


## User Events (GET /users/{uid}/events)

```
Running 10s test @ http://localhost:8080
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency   111.99ms  231.07ms   1.96s    91.76%
    Req/Sec   237.43    102.48   410.00     77.12%
  Latency Distribution
     50%   41.88ms
     75%   65.19ms
     90%  181.71ms
     99%    1.15s 
  7278 requests in 10.01s, 83.22MB read
  Socket errors: connect 0, read 0, write 0, timeout 38
Requests/sec:    727.23
Transfer/sec:      8.32MB
```


## Stats (GET /stats)

```
Running 10s test @ http://localhost:8080
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency   146.49ms  243.93ms   1.15s    88.61%
    Req/Sec    22.14     36.91   150.00     90.48%
  Latency Distribution
     50%   31.62ms
     75%  190.41ms
     90%  432.35ms
     99%    1.15s 
  127 requests in 10.01s, 24.13KB read
  Socket errors: connect 0, read 0, write 0, timeout 48
Requests/sec:     12.68
Transfer/sec:      2.41KB
```


---

**Benchmark completed at:** Thu Feb 12 10:16:20 PM MSK 2026

