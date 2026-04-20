FROM golang:1.26.2-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o bin/serenity ./main.go

FROM alpine:3.21

RUN addgroup -S serenity && adduser -S -G serenity serenity

WORKDIR /app

COPY --from=builder /app/bin/serenity .

USER serenity

EXPOSE 8080

ENTRYPOINT ["./serenity", "service"]
