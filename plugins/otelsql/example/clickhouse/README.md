# clickhouse demo

## How to run

### Docker

Start docker:

```bash
docker run -p 8123:8123 -p 9000:9000 --name some-clickhouse-server --ulimit nofile=262144:262144 clickhouse/clickhouse-server:22.8.4.7-alpine
```

Start app:

sql.Open mode:
```bash 
ENV_MODE=open go run main.go
```

connector mode:
```bash
ENV_MODE=connect go run main.go
```
