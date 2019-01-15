FROM golang:1.11

ENV APP_DIR /go/src/github.com/brianfoshee/publish

RUN mkdir -p ${APP_DIR}

WORKDIR $APP_DIR

ADD ./ ./

RUN go get ./... && go build ./cmd/publish

ENTRYPOINT ["./publish"]
