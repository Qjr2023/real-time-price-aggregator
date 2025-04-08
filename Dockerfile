FROM golang:1.23 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
RUN apt-get update && apt-get install -y curl iputils-ping net-tools
COPY . .
RUN go build -o server cmd/main.go
FROM golang:1.23
WORKDIR /app
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]