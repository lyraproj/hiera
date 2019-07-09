FROM golang:1.13-rc-alpine3.10 as build_base

WORKDIR /go/src/github.com/lyraproj/hiera
ENV GO111MODULE=on
COPY . .

RUN go install ./...

# Create a new minimalisic image that doesn't contain the build environment and
# copy the executable over
FROM alpine
COPY --from=build_base /go/bin/hieraserver /bin/hieraserver
CMD ["/bin/hieraserver", "--port", "8080"]
