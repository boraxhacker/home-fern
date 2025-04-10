FROM alpine:latest

WORKDIR /app

COPY home-fern-alpine home-fern

CMD ["/app/home-fern"]