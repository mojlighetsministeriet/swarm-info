# Run the build
FROM mojlighetsministeriet/go-polymer-faster-build
ENV WORKDIR /go/src/github.com/mojlighetsministeriet/swarm-info
COPY . $WORKDIR
WORKDIR $WORKDIR
RUN go get -t -v ./...
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build

# Create the final docker image
FROM scratch
COPY --from=0 /go/src/github.com/mojlighetsministeriet/swarm-info/client/index.html /client/index.html
COPY --from=0 /go/src/github.com/mojlighetsministeriet/swarm-info/client/node_modules/d3/build/d3.min.js /client/node_modules/d3/build/d3.min.js
COPY --from=0 /go/src/github.com/mojlighetsministeriet/swarm-info/swarm-info /
ENTRYPOINT ["/swarm-info"]
