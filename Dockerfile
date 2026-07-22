FROM golang:latest

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .
# Tracked template only; local secrets live in package/config/config.yml (gitignored) or env.
RUN if [ ! -f package/config/config.yml ]; then cp package/config/config.yml.example package/config/config.yml; fi
RUN go build -o robbo_server

EXPOSE 8080

CMD [ "/app/robbo_server" ]