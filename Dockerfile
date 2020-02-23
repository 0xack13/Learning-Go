FROM golang

ADD ./main.go ./wine-api/main.go
ADD ./winemag-data-130k-v2.csv ./wine-api/winemag-data-130k-v2.csv

WORKDIR /go/wine-api

RUN go build

ENTRYPOINT ./wine-api

EXPOSE 8080