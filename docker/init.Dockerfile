FROM k8s.gcr.io/build-image/debian-iptables:buster-v1.6.6

RUN clean-install ca-certificates
COPY ./init/init-iptables.sh /bin/
RUN chmod +x /bin/init-iptables.sh

ENTRYPOINT ["./bin/init-iptables.sh"]
