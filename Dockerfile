FROM alpine:latest

ARG CI_PROJECT_NAME="binary"

ADD $CI_PROJECT_NAME /app/$CI_PROJECT_NAME
ADD conf.yml /app/conf.d/conf.yml

WORKDIR app
RUN echo 'Running mail-seder with configuration file at "/app/conf.d/conf.yaml"'
CMD ["/app/$CI_PROJECT_NAME","-c", "/app/conf.d/conf.yaml"]
