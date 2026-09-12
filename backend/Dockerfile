FROM golang:1.26-alpine AS build

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

# Kernel 96: no USER directive meant this ran as root by default -- not
# independently exploitable given everything else this pass checked, but a
# real missing defense-in-depth layer regardless. STORAGE_ROOT/BACKUP_DIR/
# EXPORTS_ROOT are bind-mounted from the host at container-run-time (their
# actual paths aren't known at build time), owned by whatever host user ran
# generate-env.sh's mkdir -- a fixed UID here could easily not match that
# and get permission-denied on every existing install. generate-env.sh now
# makes those three directories group/other-writable specifically so any
# container UID can write them regardless of exact host ownership.
RUN adduser -D -u 65532 victory
USER 65532

EXPOSE 8081
CMD ["victory"]
