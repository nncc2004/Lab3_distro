FROM golang:1.27 AS builder

WORKDIR /app

COPY shared/ ./shared/
COPY VM2/procesadora3/ ./VM2/procesadora3/

WORKDIR /app/VM2/procesadora3

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o procesadora .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/VM2/procesadora3/procesadora .

CMD ["./procesadora"]