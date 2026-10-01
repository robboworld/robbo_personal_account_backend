# Build stage: pinned toolchain (was golang:latest, so each rebuild could change Go versions).
FROM golang:1.26 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# timetzdata: streak time zones work without tzdata in the runtime image.
RUN CGO_ENABLED=0 GOMAXPROCS=1 go build -p 1 -trimpath -buildvcs=false -tags timetzdata -o /out/robbo_server .

# Runtime stage: binary + config template + addon placeholder, no sources or toolchain.
FROM alpine:3.22

# The licensing key is bind-mounted with mode 0600; the user must own it (uid 1000 locally).
ARG APP_UID=1000
RUN apk add --no-cache ca-certificates \
    && adduser -D -H -u ${APP_UID} app

WORKDIR /app
COPY --from=build /out/robbo_server /app/robbo_server
# Image always gets the template; shop secrets come from env (PAYMENTS_YOOKASSA_*).
COPY package/config/config.yml.example /app/package/config/config.yml
# Addon placeholder (private keys are excluded by .dockerignore and mounted at runtime).
COPY keys/licensing/addon /app/keys/licensing/addon

USER app
EXPOSE 8080

CMD [ "/app/robbo_server" ]
