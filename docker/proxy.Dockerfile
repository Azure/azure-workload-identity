FROM alpine:latest
COPY bin/proxy /bin/
RUN chmod a+x /bin/proxy

ENTRYPOINT [ "proxy" ]
EXPOSE 8000
