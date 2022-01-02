# Demo

* rate(system_cpu_time{}[1m])

inside demo folder:

1. run docker-compose: `$ docker-composer up`
2. go run client: `$ go run client/main.go`
3. visit: [http://127.0.0.1:3000](http://127.0.0.1:3000)

## tempo
examples: https://github.com/grafana/tempo/tree/main/example/docker-compose/otel-collector

seach enable:
* https://github.com/grafana/tempo/tree/main/example/docker-compose/tempo-search

## loki setup
* https://grafana.com/docs/grafana/latest/datasources/loki/

### Derived fields

```
jsonData:
derivedFields:
- datasourceUid: tempo
matcherRegex: "\"traceID\":\"(\\w+)\""
name: trace
url: '$${__value.raw}'
```
