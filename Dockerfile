FROM golang:1.22-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git gcc musl-dev sqlite-dev

COPY go.mod go.sum ./
RUN go mod download

COPY main.go ./

RUN CGO_ENABLED=1 GOOS=linux go build -o whatsmeow-api main.go

FROM alpine:latest

RUN apk add --no-cache ca-certificates sqlite-libs

WORKDIR /root/

COPY --from=builder /app/whatsmeow-api .

EXPOSE 3000

CMD ["./whatsmeow-api"]
