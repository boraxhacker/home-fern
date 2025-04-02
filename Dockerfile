FROM alpine:latest

WORKDIR /app

COPY home-ssm home-ssm

CMD ["/app/home-ssm"]