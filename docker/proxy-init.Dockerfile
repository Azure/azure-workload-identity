ARG BASEIMAGE=k8s.gcr.io/build-image/debian-iptables:bullseye-v1.5.1

FROM --platform=${TARGETPLATFORM:-linux/amd64} ${BASEIMAGE}

# upgrading zlib1g due to CVE-2022-37434
# upgrading libc-bin and libc6 due to CVE-2021-3999
# upgrading libpcre2-8-0 due to CVE-2022-1586, CVE-2022-1587
RUN clean-install ca-certificates zlib1g libc-bin libc6 libpcre2-8-0
COPY ./init/init-iptables.sh /bin/
RUN chmod +x /bin/init-iptables.sh
# Kubernetes runAsNonRoot requires USER to be numeric
USER 65532:65532

ENTRYPOINT ["./bin/init-iptables.sh"]
