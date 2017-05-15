FROM golang:1.7.3
RUN go get -t github.com/rahulsom/kubehosts
CMD ["/go/bin/kubehosts"]
