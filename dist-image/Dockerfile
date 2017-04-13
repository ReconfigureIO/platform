FROM debian:jessie
RUN apt-get update && apt-get install -y ca-certificates && rm -rf /var/lib/apt/lists/*
COPY dist /
ENV GIN_MODE=release PORT=80
CMD ["/main"]