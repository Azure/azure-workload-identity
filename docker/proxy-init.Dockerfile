FROM --platform=${TARGETPLATFORM:-linux/amd64} k8s.gcr.io/build-image/debian-iptables:bullseye-v1.0.0

# upgrading libssl1.1 due to CVE-2021-3711
RUN clean-install ca-certificates libssl1.1
COPY ./init/init-iptables.sh /bin/
RUN chmod +x /bin/init-iptables.sh

ENTRYPOINT ["./bin/init-iptables.sh"]
