FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY VM3/go.mod VM3/go.sum ./
COPY shared/ ./shared/

RUN go mod edit -replace=lab3-shared=./shared
RUN go mod download

COPY VM3/nameserver/ ./nameserver/

RUN go build -o nameserver ./nameserver

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/nameserver .

EXPOSE 50051

CMD ["./nameserver"]