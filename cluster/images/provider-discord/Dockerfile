FROM gcr.io/distroless/static:nonroot

ARG TARGETOS
ARG TARGETARCH

COPY _output/bin/${TARGETOS}_${TARGETARCH}/provider /usr/local/bin/provider
COPY package.yaml /package.yaml

USER 65532:65532
ENTRYPOINT ["/usr/local/bin/provider"]
