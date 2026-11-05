FROM golang:1.27 AS builder

WORKDIR /app

COPY shared/ ./shared/
COPY VM2/procesadora2/ ./VM2/procesadora2/

WORKDIR /app/VM2/procesadora2

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o procesadora .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/VM2/procesadora2/procesadora .

CMD ["./procesadora"]