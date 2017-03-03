FROM golang:1.7.4
RUN apt-get update && apt-get install -y
RUN curl https://glide.sh/get | sh
RUN go get github.com/gin-gonic/gin
RUN wget https://raw.githubusercontent.com/gin-gonic/gin/master/examples/basic/main.go
RUN go run main.go
VOLUME /go