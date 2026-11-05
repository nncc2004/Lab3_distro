FROM golang:1.27 AS builder

WORKDIR /app

COPY shared/ ./shared/
COPY VM4/procesadora/ ./VM4/procesadora/

WORKDIR /app/VM4/procesadora

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o procesadora .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/VM4/procesadora/procesadora .

CMD ["./procesadora"]