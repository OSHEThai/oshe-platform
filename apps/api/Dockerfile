# Pinned Go toolchain builder stage
FROM golang:1.26.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS builder

WORKDIR /src

# Copy go.mod files for dependency caching
COPY apps/api/go.mod apps/api/
COPY modules/organization-tenancy/go.mod modules/organization-tenancy/

# Copy sources required for build
COPY modules/organization-tenancy modules/organization-tenancy
COPY apps/api apps/api

WORKDIR /src/apps/api

# Build static Linux binary
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /bin/oshe-api ./cmd/oshe-api

# Pinned Alpine runtime stage
FROM alpine:3.21@sha256:48b0309ca019d89d40f670aa1bc06e426dc0931948452e8491e3d65087abc07d

RUN apk add --no-cache ca-certificates tzdata wget

WORKDIR /app

COPY --from=builder /bin/oshe-api /usr/local/bin/oshe-api

ENV PORT=8080 \
    OSHE_AUTH_MODE=fail_closed

EXPOSE 8080

HEALTHCHECK --interval=10s --timeout=5s --retries=5 \
  CMD ["wget", "--no-verbose", "--spider", "http://127.0.0.1:8080/health"]

USER nobody:nobody

ENTRYPOINT ["/usr/local/bin/oshe-api"]
