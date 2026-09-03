# Synthetic-only Go/Node/Python development container.
# Version-pinned to the ADR-0006 toolchain (Go 1.26.5, Node 24.20.0, pnpm 11.24.0, Python 3.14.7).
FROM golang:1.26.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS go
FROM node:24.20.0-alpine@sha256:e67514e5d0f6c46656005e1b693b2ec9d52e80b641307de684d4a015ba7a4eaf AS node

FROM python:3.14.7-alpine@sha256:c6ead215bfd31f1e433d968853b7a769989117115b728874824e6c0a27cb96fc

RUN addgroup -g 1000 vscode && \
    adduser -u 1000 -G vscode -s /bin/sh -D vscode

# Install Go toolchain (copied from the pinned go stage).
COPY --from=go /usr/local/go /usr/local/go
ENV PATH="/usr/local/go/bin:${PATH}"


# Install Node + corepack/pnpm (copied from the pinned node stage).
COPY --from=node /usr/local/lib/node_modules /usr/local/lib/node_modules
COPY --from=node /usr/local/bin/node /usr/local/bin/node
COPY --from=node /usr/local/bin/corepack /usr/local/bin/corepack
ENV PATH="/usr/local/bin:${PATH}"
RUN corepack enable && corepack prepare pnpm@11.24.0 --activate

RUN apk add --no-cache git

WORKDIR /workspaces/oshe-platform
