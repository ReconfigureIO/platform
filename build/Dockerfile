FROM golang:1.7.4
RUN apt-get update && apt-get install -y
RUN curl https://glide.sh/get | sh
RUN go get github.com/gin-gonic/gin
COPY ./main.go ~/