# Go-Collider Benchmark Report - json_optimization

**Date:** Sat Feb 28 07:08:49 PM MSK 2026
**Configuration:**
- Host: http://localhost:8080
- Benchmark Name: json_optimization
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
    Latency    26.38ms   27.07ms 135.01ms   78.49%
    Req/Sec   628.53    478.98     2.85k    89.00%
  Latency Distribution
     50%    9.87ms
     75%   46.83ms
     90%   70.63ms
     99%   86.35ms
  25043 requests in 10.01s, 7.23MB read
Requests/sec:   2502.06
Transfer/sec:    740.09KB
```


## Events List (GET /events)

```
Running 10s test @ http://localhost:8080/events?page=1&limit=100
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    25.70ms   27.92ms 135.69ms   79.47%
    Req/Sec   724.32    362.07     2.63k    80.86%
  Latency Distribution
     50%    7.36ms
     75%   47.16ms
     90%   70.22ms
     99%   87.72ms
  29082 requests in 10.10s, 479.18MB read
Requests/sec:   2879.75
Transfer/sec:     47.45MB
```


## User Events (GET /users/{uid}/events)

```
Running 10s test @ http://localhost:8080
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    23.18ms   26.64ms  99.98ms   78.26%
    Req/Sec     0.96k   607.94     2.91k    83.00%
  Latency Distribution
     50%    5.56ms
     75%   42.27ms
     90%   67.58ms
     99%   83.48ms
  38083 requests in 10.00s, 254.57MB read
Requests/sec:   3806.41
Transfer/sec:     25.44MB
```


## Stats (GET /stats)

```
Running 10s test @ http://localhost:8080
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    26.36ms   28.32ms 246.97ms   78.39%
    Req/Sec   676.59    285.46     2.52k    94.50%
  Latency Distribution
     50%    6.66ms
     75%   45.43ms
     90%   72.24ms
     99%   86.69ms
  26946 requests in 10.01s, 4.37MB read
Requests/sec:   2692.22
Transfer/sec:    446.94KB
```


---

**Benchmark completed at:** Sat Feb 28 07:09:29 PM MSK 2026

