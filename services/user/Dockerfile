FROM golang:alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o userservice ./cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/userservice .
CMD ["./userservice"] 