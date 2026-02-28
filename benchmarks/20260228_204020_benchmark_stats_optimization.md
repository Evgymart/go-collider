# Go-Collider Benchmark Report - stats_optimization

**Date:** Sat Feb 28 08:40:20 PM MSK 2026
**Configuration:**
- Host: http://localhost:8080
- Benchmark Name: stats_optimization
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
    Latency    27.35ms   28.49ms 102.81ms   77.85%
    Req/Sec   627.63    523.52     2.87k    90.50%
  Latency Distribution
     50%    8.81ms
     75%   50.17ms
     90%   74.68ms
     99%   88.78ms
  25008 requests in 10.01s, 7.22MB read
Requests/sec:   2499.16
Transfer/sec:    739.22KB
```


## Events List (GET /events)

```
Running 10s test @ http://localhost:8080/events?page=1&limit=100
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    25.96ms   27.95ms 129.57ms   78.88%
    Req/Sec   707.42    412.20     2.67k    88.16%
  Latency Distribution
     50%    7.62ms
     75%   46.92ms
     90%   71.46ms
     99%   87.07ms
  28389 requests in 10.10s, 467.76MB read
Requests/sec:   2811.06
Transfer/sec:     46.32MB
```


## User Events (GET /users/{uid}/events)

```
Running 10s test @ http://localhost:8080
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    23.47ms   26.78ms  90.30ms   78.15%
    Req/Sec     0.92k   472.92     2.97k    82.59%
  Latency Distribution
     50%    5.31ms
     75%   43.04ms
     90%   67.64ms
     99%   83.93ms
  36938 requests in 10.10s, 246.92MB read
Requests/sec:   3657.45
Transfer/sec:     24.45MB
```


## Stats (GET /stats)

```
Running 10s test @ http://localhost:8080
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    23.42ms   27.09ms 165.53ms   78.51%
    Req/Sec     0.95k   700.84     4.77k    84.00%
  Latency Distribution
     50%    5.71ms
     75%   42.93ms
     90%   68.35ms
     99%   84.61ms
  37920 requests in 10.00s, 6.15MB read
Requests/sec:   3790.32
Transfer/sec:    629.25KB
```


---

**Benchmark completed at:** Sat Feb 28 08:41:00 PM MSK 2026

