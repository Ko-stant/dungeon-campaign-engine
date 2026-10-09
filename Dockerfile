# syntax=docker/dockerfile:1
#
# The app as one container (docs/ONLINE_AND_RULES_PLAN.md, Phase 6).
#
# HeroQuest art is not in this repository. `make image` passes it in as the
# named build context "assets"; without it the build fails. The catalog
# (classes, monsters, furniture, traps) is not in the image: it is in the
# database (make import-content). An image built this way is for the GM's group
# only: push it only to the private registry (DOCKER_IMAGE), never to GitHub or
# anywhere public.
#
# Run exactly one container: session locks and live connections are kept in
# memory. Migrations are built into the binary and run on start.

# The web bundle and CSS are the same on every platform: build them natively.
FROM --platform=$BUILDPLATFORM oven/bun:1.4.2 AS web
WORKDIR /src
COPY package.json bun.lock ./
RUN bun install --frozen-lockfile
COPY tsconfig.json ./
COPY scripts/build.ts scripts/
COPY internal/web ./internal/web
RUN bun run build:web && bun run tailwind:build

# Cross-compile the server for the target platform.
FROM --platform=$BUILDPLATFORM golang:1.27 AS server
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY cmd ./cmd
COPY db ./db
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/dungeon-campaign-engine ./cmd/server

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /app
COPY --from=server /out/dungeon-campaign-engine ./
COPY --from=web /src/internal/web/static ./internal/web/static
# Only the board tiles the app draws (the cleaned ones; card art and scans stay
# out). The catalog is in the database.
COPY --from=assets /tiles_cleaned ./assets/tiles_cleaned
ENV PORT=8080
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app/dungeon-campaign-engine"]
