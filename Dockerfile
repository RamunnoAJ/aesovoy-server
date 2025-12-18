FROM golang:1.24-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git
COPY go.* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o main .

FROM alpine:3.20
WORKDIR /app
RUN apk add --no-cache ca-certificates tzdata
ENV PROJECT_ROOT=/app
COPY --from=builder /app/main .
COPY docs/ ./docs/
RUN mkdir -p facturas uploads/expenses

EXPOSE 8080
ENTRYPOINT ["./main"]
