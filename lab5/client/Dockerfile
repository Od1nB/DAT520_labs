# Start from a image with the latest version of Go installed
# and a workspace (GOPATH) configured at /go.
FROM golang

# Copy the local package files (lab5) to the container's workspace.
COPY . /go/src/app

WORKDIR /go/src/app

#Build and install your application inside the container.
RUN go install -v ./lab5/client/app.go

ENTRYPOINT ["/go/bin/app"]