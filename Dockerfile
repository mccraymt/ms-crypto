# FROM golang:onbuild (doesn't work on circleci)
# FROM golang
FROM golang:1.6.2

WORKDIR /go/src/github.com/mccraymt/ms-crypto
ADD . /go/src/github.com/mccraymt/ms-crypto/

RUN go get github.com/tools/godep
RUN godep restore

RUN go install github.com/mccraymt/ms-crypto

CMD /go/bin/ms-crypto

EXPOSE 4000
