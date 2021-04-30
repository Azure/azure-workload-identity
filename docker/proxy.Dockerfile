FROM alpine:latest
COPY ./_output/proxy /bin/
RUN chmod a+x /bin/proxy

ENTRYPOINT [ "proxy" ]
EXPOSE 8000
