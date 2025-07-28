FROM gcr.io/distroless/static:nonroot
COPY provider /usr/local/bin/provider
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["provider"]