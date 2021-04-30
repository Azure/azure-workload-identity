FROM alpine:latest

RUN apk update && apk add iptables
COPY ./init/init-iptables.sh /bin/
RUN chmod +x /bin/init-iptables.sh

ENTRYPOINT ["./bin/init-iptables.sh"]
