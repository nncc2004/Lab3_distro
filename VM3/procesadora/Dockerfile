FROM golang:1.27 AS builder

WORKDIR /app

COPY shared/ ./shared/
COPY VM3/procesadora/ ./VM3/procesadora/

WORKDIR /app/VM3/procesadora

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o procesadora .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/VM3/procesadora/procesadora .

CMD ["./procesadora"]