FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY shared/ ./shared/
COPY VM2/agente/ ./VM2/agente/

WORKDIR /app/VM2/agente

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o agente .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/VM2/agente/agente .

CMD ["./agente"]