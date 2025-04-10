FROM alpine:latest

WORKDIR /app

COPY home-fern home-fern

CMD ["/app/home-fern"]