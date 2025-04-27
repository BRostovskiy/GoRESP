FROM --platform=linux/arm64 golang:1.21-alpine as builder

WORKDIR /opt
COPY . .

RUN go build -o goredis cmd/resp-server/main.go

FROM --platform=linux/arm64 golang:1.21-alpine as builder2

WORKDIR /opt

COPY cmd/client/main.go .
COPY go.mod .
COPY go.sum .

RUN go build -o goredisclient

FROM alpine:latest
COPY --from=builder /opt/goredis ./goredis
COPY --from=builder2 /opt/goredisclient ./goredisclient

CMD ["./goredis"]