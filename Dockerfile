# Build stage: pinned toolchain (was golang:latest, so each rebuild could change Go versions).
FROM golang:1.26 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# timetzdata: streak time zones work without tzdata in the runtime image.
# COPY above invalidates this layer on any source change; the cache mount keeps compiled packages.
# A parallel cold compile needs 4 GB: at 3 GB it gets OOM-killed. With less free memory pass
# --build-arg GOMAXPROCS=1 (go build takes -p from it): 2 GB is enough, 2-3 times slower.
ARG GOMAXPROCS
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -buildvcs=false -tags timetzdata -o /out/robbo_server .

# Runtime stage: binary + config template + addon placeholder, no sources or toolchain.
FROM alpine:3.22

# The licensing key is bind-mounted with mode 0600; the user must own it (uid 1000 locally).
ARG APP_UID=1000
RUN apk add --no-cache ca-certificates \
    && adduser -D -H -u ${APP_UID} app \
    && mkdir -p /app/data/userfiles \
    && chown ${APP_UID}:${APP_UID} /app/data/userfiles

WORKDIR /app
COPY --from=build /out/robbo_server /app/robbo_server
# Image always gets the template; shop secrets come from env (PAYMENTS_YOOKASSA_*).
COPY package/config/config.yml.example /app/package/config/config.yml
# Addon placeholder (private keys are excluded by .dockerignore and mounted at runtime).
COPY keys/licensing/addon /app/keys/licensing/addon

USER app
EXPOSE 8080
# Liveness only; /readyz (databases) is for load balancers and deploy checks.
HEALTHCHECK --interval=30s --timeout=3s --start-period=20s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1

CMD [ "/app/robbo_server" ]
