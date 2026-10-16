FROM golang:1.27.2-trixie

WORKDIR /app

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o cake cmd/main.go

EXPOSE 8080

CMD ["./cake"]