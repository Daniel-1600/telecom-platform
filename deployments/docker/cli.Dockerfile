FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY apps/cli/go.mod apps/cli/go.sum ./
RUN go mod download

COPY apps/cli/ ./

RUN CGO_ENABLED=0 GOOS=linux go build -o cli .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/cli .

CMD ["./cli"]
