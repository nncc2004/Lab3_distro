FROM golang:1.27

WORKDIR /app

# Copia el modulo compartido
COPY shared ./shared

# Copia el agente de prueba
COPY VM2/prueba_agente ./VM2/prueba_agente

WORKDIR /app/VM2/prueba_agente

# Descarga las dependencias
RUN go mod download

# Compila el agente
RUN go build -o agente-prueba .

CMD ["./agente-prueba"]