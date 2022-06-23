# Hot R.O.D. - Rides on Demand

Project taken from https://github.com/jaegertracing/jaeger/blob/main/examples/hotrod/README.md

This is a demo application that consists of several microservices and illustrates
the use of the OpenTracing API. It can be run standalone, but requires Jaeger backend
to view the traces. A tutorial / walkthrough is available:
* as a blog post [Take OpenTracing for a HotROD ride][hotrod-tutorial],
* as a video [OpenShift Commons Briefing: Distributed Tracing with Jaeger & Prometheus on Kubernetes][hotrod-openshift].

## Features

* Discover architecture of the whole system via data-driven dependency diagram
* View request timeline & errors, understand how the app works
* Find sources of latency, lack of concurrency
* Highly contextualized logging
* Use baggage propagation to
    * Diagnose inter-request contention (queueing)
    * Attribute time spent in a service
* Use open source libraries with OpenTracing integration to get vendor-neutral instrumentation for free

## Running

### Run TEL infra

* Download folder `demo` from https://github.com/d7561985/tel/tree/master/example
* Run Tel infrastructure backend (OTEL Collector, Grafana, Grafana Loki, Grafana Tempo and Prometheus) and HotROD demo with `docker-compose -f path-to-yml-file up`
* Access Grafana UI at http://localhost:3000
* Shutdown / cleanup with `docker-compose -f path-to-yml-file down`

Alternatively, you can run each component separately as described below.

### Run manually Hot R.O.D

```bash
go run ./main.go all
```

* Access HotROD app at http://localhost:8080