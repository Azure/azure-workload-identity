FROM k8s.gcr.io/build-image/debian-iptables:buster-v1.6.6

# upgrading libssl1.1 due to CVE-2021-33910 and CVE-2021-3712
RUN clean-install ca-certificates libssl1.1
COPY ./init/init-iptables.sh /bin/
RUN chmod +x /bin/init-iptables.sh

ENTRYPOINT ["./bin/init-iptables.sh"]
