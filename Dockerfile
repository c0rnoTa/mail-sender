FROM alpine:latest

ADD mail-sender /app/mail-sender
ADD conf.yml /app/conf.d/conf.yml

WORKDIR app
CMD ["/app/mail-sender","-c", "/app/conf.d/conf.yaml"]
