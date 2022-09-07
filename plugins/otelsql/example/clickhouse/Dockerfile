FROM golang:alpine AS base
COPY . /src/example
WORKDIR /src/example
RUN go install main.go

FROM base
COPY --from=base /go/bin/main /go/bin/main
CMD ["/go/bin/main"]