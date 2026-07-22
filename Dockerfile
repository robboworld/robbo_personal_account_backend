FROM golang:latest

WORKDIR /app

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .
# Image always gets the template; shop secrets come from env (PAYMENTS_YOOKASSA_*).
RUN cp package/config/config.yml.example package/config/config.yml
RUN go build -o robbo_server

EXPOSE 8080

CMD [ "/app/robbo_server" ]