FROM golang:1.18-alpine

COPY . /build
WORKDIR /build

RUN go build -o /app /build

CMD /app