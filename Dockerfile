# syntax=docker/dockerfile:1
FROM golang:1.21-alpine as builder
RUN apk add --update git gcc g++ openssh-client make
WORKDIR /app
COPY go.mod go.sum ./
COPY go-relay-client/ go-relay-client/
RUN make dependencies
RUN make build

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /
COPY --from=builder /app/build/goteth ./
COPY --from=builder /app/pkg/db/migrations ./pkg/db/migrations
ENTRYPOINT ["/goteth"]
