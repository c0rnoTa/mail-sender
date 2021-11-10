FROM alpine:latest

ARG CI_PROJECT_NAME="binary"

ADD $CI_PROJECT_NAME /app/$CI_PROJECT_NAME
ADD ./conf.yml /app/

WORKDIR app

ENTRYPOINT ["/app/$CI_PROJECT_NAME","-c", "/app/conf.yaml"]
