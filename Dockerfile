# ---------- Stage 1: Build ----------
FROM golang:1.24-alpine AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o expense-server ./cmd/server

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/expense-server .
EXPOSE 8091 8092

CMD ["./expense-server"]