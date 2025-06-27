FROM golang:alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o linkservice ./cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/linkservice .
CMD ["./linkservice"] 