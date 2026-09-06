FROM golang:1.25-alpine AS build

WORKDIR /app

COPY . .
RUN go mod tidy
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /victory ./cmd/victory
RUN CGO_ENABLED=0 GOOS=linux go build -o /victory-bootstrap ./cmd/victory-bootstrap

FROM alpine:3.20
WORKDIR /app
# postgresql16-client provides pg_dump for the Kernel 72 pre-migration
# auto-backup; it must match the postgres:16 server major version.
RUN apk add --no-cache postgresql16-client
COPY --from=build /victory /usr/local/bin/victory
# victory-bootstrap is never the container's own entrypoint -- it's invoked
# via `podman exec` / `docker exec` by whatever drives first-run setup (the
# Windows installer, or generate-env.sh's caller during development) to
# grant the first account a producer role. Kernel 100: that has to happen
# without ever handing the end user a terminal, so the binary needs to be
# reachable inside the running container, not run from a human's shell.
COPY --from=build /victory-bootstrap /usr/local/bin/victory-bootstrap

EXPOSE 8081
CMD ["victory"]
