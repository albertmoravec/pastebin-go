FROM golang:1-alpine

RUN echo $'\n\
  #!/bin/sh\n\
  rm -f config.json\n\
  \n\
  echo \"{\"appName\": \"'${APP_NAME:-Pastebin}'\", \"appUrl\": \"'${APP_URL:-http://localhost:8080}'\",\"httpPort\": \"'${PORT:-8080}'\",\"redisHost\": \"'${REDIS_HOST:-127.0.0.1:6379}'\",\"redisPass\": \"'$REDIS_PASS'\"}\" > config.json\n\
  \n\
  sh -c ./pastebin' > run.sh && chmod +x run.sh

CMD ./run.sh