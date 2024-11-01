FROM gcr.io/distroless/static-debian12

ARG TARGETOS
ARG TARGETARCH

COPY aws-checker /usr/local/bin/aws-checker

EXPOSE 8080

CMD ["/usr/local/bin/aws-checker"]
