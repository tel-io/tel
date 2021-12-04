# Demo


* rate(system_cpu_time{}[1m])


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
