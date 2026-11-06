# Etapa de compilación
FROM golang:1.27-alpine AS builder

WORKDIR /app

# Copiar el módulo compartido
COPY shared/ ./shared/

# Copiar el código de Policía
COPY VM4/policia/ ./VM4/policia/

# Compilar
WORKDIR /app/VM4/policia

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /policia .

# Etapa de ejecución
FROM alpine:latest

WORKDIR /app

COPY --from=builder /policia .

CMD ["./policia"]