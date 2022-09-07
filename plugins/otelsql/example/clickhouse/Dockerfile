FROM golang:alpine AS base
COPY . /src/example
WORKDIR /src/example

FROM base
RUN go install main.go
CMD ["/go/bin/main"]