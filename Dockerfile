FROM golang:latest as builder
WORKDIR /code
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /code/app main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /code/app /usr/bin/app
CMD ["/usr/bin/app"]
