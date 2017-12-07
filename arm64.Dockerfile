# Run the build
FROM mojlighetsministeriet/go-polymer-faster-build
ENV WORKDIR /go/src/github.com/mojlighetsministeriet/swarm-info
COPY . $WORKDIR
WORKDIR $WORKDIR
RUN go get -t -v ./...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build

# Create the final docker image
FROM scratch
COPY --from=0 /go/src/github.com/mojlighetsministeriet/swarm-info/swarm-info /
ENTRYPOINT ["/swarm-info"]