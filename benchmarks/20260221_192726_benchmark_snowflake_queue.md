# Go-Collider Benchmark Report - snowflake_queue

**Date:** Sat Feb 21 07:27:26 PM MSK 2026
**Configuration:**
- Host: http://localhost:8080
- Benchmark Name: snowflake_queue
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
    Latency    20.25ms   26.27ms 203.21ms   80.49%
    Req/Sec     1.58k   812.87     6.51k    79.05%
  Latency Distribution
     50%    3.58ms
     75%   37.32ms
     90%   63.11ms
     99%   89.93ms
  63080 requests in 10.10s, 18.34MB read
Requests/sec:   6245.50
Transfer/sec:      1.82MB
```


## Events List (GET /events)

```
Running 10s test @ http://localhost:8080/events?page=1&limit=100
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency     1.20s   313.69ms   1.64s    56.76%
    Req/Sec     8.09      6.25    30.00     76.42%
  Latency Distribution
     50%    1.09s 
     75%    1.55s 
     90%    1.60s 
     99%    1.64s 
  164 requests in 10.01s, 2.70MB read
  Socket errors: connect 0, read 0, write 0, timeout 127
Requests/sec:     16.38
Transfer/sec:    276.33KB
```


## User Events (GET /users/{uid}/events)

```
Running 10s test @ http://localhost:8080
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency   106.93ms  224.95ms   1.61s    91.48%
    Req/Sec   318.31     58.91   414.00     83.53%
  Latency Distribution
     50%   38.72ms
     75%   51.97ms
     90%  237.34ms
     99%    1.15s 
  10918 requests in 10.01s, 73.02MB read
Requests/sec:   1090.83
Transfer/sec:      7.30MB
```


## Stats (GET /stats)

```
Running 10s test @ http://localhost:8080
  4 threads and 50 connections
  Thread Stats   Avg      Stdev     Max   +/- Stdev
    Latency    74.54ms   55.71ms 359.38ms   61.66%
    Req/Sec   167.39     25.27   250.00     69.75%
  Latency Distribution
     50%   64.08ms
     75%  118.18ms
     90%  151.91ms
     99%  205.16ms
  6677 requests in 10.01s, 1.08MB read
Requests/sec:    666.91
Transfer/sec:    110.68KB
```


---

**Benchmark completed at:** Sat Feb 21 07:28:07 PM MSK 2026

