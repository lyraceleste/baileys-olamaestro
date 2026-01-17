FROM golang:1.22-alpine AS builder

WORKDIR /app

# Instalar dependências
RUN apk add --no-cache git gcc musl-dev

# Copiar código
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build
RUN CGO_ENABLED=1 go build -o whatsmeow-api .

# Imagem final
FROM alpine:latest

RUN apk add --no-cache ca-certificates

WORKDIR /root/

COPY --from=builder /app/whatsmeow-api .

EXPOSE 3000

CMD ["./whatsmeow-api"]
