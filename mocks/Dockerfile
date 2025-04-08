FROM golang:1.23 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
RUN apt-get update && apt-get install -y curl iputils-ping net-tools
COPY . .
RUN go build -o mock_server mocks/mock_server.go
FROM golang:1.23
WORKDIR /app
COPY --from=builder /app/mock_server .
EXPOSE 8081 8082 8083
CMD ["./mock_server"]