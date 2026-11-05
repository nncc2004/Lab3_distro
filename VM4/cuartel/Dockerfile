FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY shared/ /shared/
COPY VM4/cuartel/go.mod VM4/cuartel/go.sum ./

RUN go mod download

COPY VM4/cuartel/ ./

RUN go build -o cuartel .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/cuartel .

CMD ["./cuartel"]