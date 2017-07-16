FROM golang:1.8.3
RUN go get -t github.com/rahulsom/kubehosts
CMD ["/go/bin/kubehosts"]
