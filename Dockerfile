FROM alpine:latest

WORKDIR /app

COPY release/home-fern-alpine home-fern

CMD ["/app/home-fern"]