FROM golang:1.25-alpine AS build

WORKDIR /app

COPY . .
RUN go mod tidy
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /victory ./cmd/victory

FROM alpine:3.20
WORKDIR /app
# postgresql16-client provides pg_dump for the Kernel 72 pre-migration
# auto-backup; it must match the postgres:16 server major version.
RUN apk add --no-cache postgresql16-client
COPY --from=build /victory /usr/local/bin/victory

EXPOSE 8081
CMD ["victory"]
