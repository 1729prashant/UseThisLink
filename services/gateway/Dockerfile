FROM golang:alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o gateway ./cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/gateway .
CMD ["./gateway"] 