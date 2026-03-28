FROM golang:1.25-alpine AS build

WORKDIR /app

COPY . .
RUN go mod tidy
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /victory ./cmd/victory

FROM alpine:3.20
WORKDIR /app
COPY --from=build /victory /usr/local/bin/victory

EXPOSE 8081
CMD ["victory"]
