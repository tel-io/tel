# demo

## Running

### Run TEL infra

* Download  or move to folder `docker` at  https://github.com/d7561985/tel/docker
* Run Tel infrastructure backend (OTEL Collector, Grafana, Grafana Loki, Grafana Tempo and Prometheus) and HotROD demo with `docker-compose -f path-to-yml-file up`
* Access Grafana UI at http://localhost:3000
* Shutdown / cleanup with `docker-compose -f path-to-yml-file down`

Alternatively, you can run each component separately as described below.

### Run manually Hot R.O.D

```bash
go run ./client/main.go
go run ./gin/main.go
```

* visit Grafana UI at http://localhost:3000 and explore metrics