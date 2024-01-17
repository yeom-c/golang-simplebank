FROM golang:1.21.6-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o main main.go \
    && apk --no-cache add curl \
    && curl -L https://github.com/golang-migrate/migrate/releases/download/v4.17.0/migrate.linux-amd64.tar.gz | tar xvz

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/migrate ./migrate
COPY .env start.sh wait-for.sh ./
COPY db/migration ./migration/

EXPOSE 8080

ENTRYPOINT ["/app/start.sh"]
CMD ["./main"]