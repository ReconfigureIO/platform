FROM debian:jessie
RUN apt-get update && apt-get install -y ca-certificates
COPY ./main /main
ENV GIN_MODE=release PORT=80
CMD ["/main"]