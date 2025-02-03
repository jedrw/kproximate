FROM --platform=$BUILDPLATFORM golang:1.22-alpine3.18 AS build
RUN apk add upx
WORKDIR $GOPATH/src/kproximate
COPY . .
ARG COMPONENT
ARG TARGETARCH
RUN cd kproximate/$COMPONENT && GOOS=linux GOARCH=$TARGETARCH go build -v -o /go/bin/$COMPONENT
RUN upx /go/bin/$COMPONENT

# final stage
FROM --platform=$BUILDPLATFORM alpine
ARG COMPONENT
RUN adduser \    
    --disabled-password \    
    --gecos "" \    
    --home "/nonexistent" \    
    --shell "/sbin/nologin" \    
    --no-create-home \    
    --uid "1001" \    
    "kproximate"

COPY --from=build /go/bin/$COMPONENT /usr/local/bin/
USER kproximate:kproximate
ENTRYPOINT ["$COMPONENT"]