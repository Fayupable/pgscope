# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# CGO disabled: pgscope has no C dependencies, and a static binary lets the
# final stage be a distroless image with no shell, no package manager, and
# nothing beyond what's strictly needed to run the binary.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /pgscope ./cmd/pgscope

RUN mkdir -p /data

# distroless/static includes CA certificates (needed if PGSCOPE_DATABASE_URL
# ever uses sslmode=verify-full against a cert-issuing Postgres) and nothing
# else: no shell, no package manager, drastically smaller attack surface
# than a full alpine or debian runtime image.
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app
COPY --from=builder /pgscope /app/pgscope
COPY --chown=nonroot:nonroot --from=builder /data /data

USER nonroot:nonroot

EXPOSE 8090

ENTRYPOINT ["/app/pgscope"]