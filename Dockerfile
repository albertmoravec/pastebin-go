FROM golang:1-alpine AS build

WORKDIR $GOPATH/src/github.com/albru123/pastebin-go

COPY . .

RUN apk update &&\
  apk add --no-cache git &&\
  go get -d -v ./... &&\
  go build pastebin.go &&\
  apk del git &&\
  mkdir /release &&\
  cp pastebin /release/ &&\
  mkdir /release/assets &&\
  cp -r assets/public /release/assets/ &&\
  cp -r templates /release/ &&\
  echo $'\n\
  #!/bin/sh\n\
  rm -f config.json\n\
  \n\
  echo \"{\\"appName\\": \\"${APP_NAME:-Pastebin}\\", \\"appUrl\\": \\"${APP_URL:-http://localhost:8080}\\",\\"httpPort\\": \\"${PORT:-8080}\\",\\"redisHost\\": \\"${REDIS_HOST:-127.0.0.1:6379}\\",\\"redisPass\\": \\"$REDIS_PASS\\"}\" > config.json\n\
  \n\
  sh -c ./pastebin' > run.sh &&\
  chmod +x run.sh &&\
  cp run.sh /release/

FROM alpine

COPY --from=build /release/ /app/
WORKDIR /app

CMD ./run.sh