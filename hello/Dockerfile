FROM golang:1.21.0-alpine3.18 AS builder

WORKDIR /usr/src/app

COPY . .
RUN go build -v -o app .

FROM alpine:3.18

WORKDIR /usr/local/bin

COPY --from=builder /usr/src/app/app .
CMD ["./app"]
