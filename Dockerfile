FROM golang


# Copy the local package files to the container's workspace.
ADD . /go/src/github.com/aprosvetova/silencebot

# Build the silencebot inside the container.
RUN go get -v github.com/aprosvetova/silencebot && go install github.com/aprosvetova/silencebot

# Run the silencebot by default when the container starts.
ENTRYPOINT /go/bin/silencebot -t 123456789:XXXxXxxXxxx0xxxXX00XXXX0XXxXXxxXxxx
