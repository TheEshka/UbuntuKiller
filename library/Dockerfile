FROM golang:1.16 AS builder

COPY . /app
WORKDIR /app
RUN go build -o service ./cmd/service


FROM debian

COPY --from=builder /app/service /app/service
CMD ["/app/service"]