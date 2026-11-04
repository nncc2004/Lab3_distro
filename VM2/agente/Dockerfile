FROM golang:1.27-alpine
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN go build -o agente .
CMD ["./agente"]