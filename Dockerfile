FROM golang

ADD ./winemag-data-130k-v2.csv ./wine-api/winemag-data-130k-v2.csv
ADD ./main.go ./wine-api/main.go

WORKDIR /go/wine-api

RUN go build

ENTRYPOINT ./wine-api

EXPOSE 8080