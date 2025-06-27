FROM golang:alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o analyticsservice ./cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/analyticsservice .
CMD ["./analyticsservice"] 