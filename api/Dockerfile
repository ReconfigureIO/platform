FROM debian:jessie
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY ./main /main
COPY ./deploy_schema /deploy_schema
ENV GIN_MODE=release PORT=80
CMD ["/main"]