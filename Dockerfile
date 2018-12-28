FROM golang:1.11-alpine

RUN apk --no-cache add git

COPY dummy_streamer.go /home/
WORKDIR /home

# added `OR echo` to ignore non-zero exit code
RUN GOPATH=/home go get ./... || echo

ENV GOPATH=/home

ENTRYPOINT ["go", "run", "dummy_streamer.go"]
CMD ["--server", "http://hds:8085"]
