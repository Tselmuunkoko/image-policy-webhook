FROM golang:1.21.0-alpine3.18 AS builder

WORKDIR /usr/src

COPY src .
RUN go build -v -o app .

FROM docker:24.0.5-dind-alpine3.18

WORKDIR /usr/local/bin

COPY --from=builder /usr/src/app .
COPY run.sh .
RUN chmod +x run.sh
ENTRYPOINT ["./run.sh"]
