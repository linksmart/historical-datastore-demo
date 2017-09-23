FROM golang:1.8-alpine

COPY dummy_streamer.go /home
WORKDIR /home

ENTRYPOINT ["go", "run", "dummy_streamer.go"]
CMD ["--server", "http://hds:8085"]
