FROM golang:1.8-alpine

ENV Ver 0.3

RUN apk add --no-cache git

WORKDIR /home

RUN git clone https://gist.github.com/b774a56091a55f8892edd9571f40d6ea.git .

ENTRYPOINT ["go", "run", "hds_dummy_stream.go"]
CMD ["--server", "http://hds:8085"]
