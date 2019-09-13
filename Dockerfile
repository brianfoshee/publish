FROM golang:1.13-alpine

LABEL "com.github.actions.name"="Publish"
LABEL "com.github.actions.description"="Build this project so others can use it in actions"
LABEL "com.github.actions.icon"="printer"
LABEL "com.github.actions.color"="yellow"

LABEL "repository"="https://github.com/brianfoshee/publish"
LABEL "homepage"="https://github.com/brianfoshee/publish"
LABEL "maintainer"="Brian Foshee <brian@brianfoshee.com>"

ENV APP_DIR /app

RUN mkdir -p ${APP_DIR}

ADD ./ ${APP_DIR}

RUN cd ${APP_DIR} && go mod download && go install ./cmd/publish

ENTRYPOINT ["publish"]
