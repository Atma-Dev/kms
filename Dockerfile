# syntax=docker/dockerfile:1.7

FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -o main main.go

# The reference seeder and the one-off data migration. They run by hand on the
# server, and the final image has no Go toolchain, so building them here is the
# only way they can run there at all.
RUN CGO_ENABLED=0 go build -o seed ./cmd/seed \
 && CGO_ENABLED=0 go build -o sync-permission ./cmd/sync-permission \
 && CGO_ENABLED=0 go build -o migrate ./cmd/migrate

FROM alpine:3.20

RUN apk add --no-cache ca-certificates curl

WORKDIR /app

COPY --from=builder /go/bin/migrate /usr/local/bin/migrate
COPY --from=builder /app/main ./main
COPY --from=builder /app/seed ./seed
COPY --from=builder /app/sync-permission ./sync-permission
COPY --from=builder /app/migrate ./migrate-data
COPY --from=builder /app/database ./database
COPY --from=builder /app/data_migration ./data_migration

EXPOSE 8000

CMD ["./main"]
