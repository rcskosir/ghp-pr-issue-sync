# syntax=docker/dockerfile:1

FROM golang:1.18-alpine

RUN apk update && apk upgrade && apk add --update alpine-sdk && \
    apk add --no-cache bash git openssh make cmake

WORKDIR /app

COPY . .

RUN make install

CMD [ "/ghp-pr-sync" ]