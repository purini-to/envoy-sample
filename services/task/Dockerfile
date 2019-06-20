FROM alpine

RUN apk --no-cache --update add tzdata

WORKDIR /app
COPY main bin/app

EXPOSE 8080

CMD ["bin/app"]
