FROM --platform=${TARGETPLATFORM:-linux/amd64} k8s.gcr.io/build-image/debian-iptables:bullseye-v1.3.0

# upgrading zlib1g due to CVE-2018-25032
# upgrading gzip and liblzma5 due to CVE-2022-1271
# upgrading libssl1.1 due to CVE-2022-1292
# upgrading dpkg due to CVE-2022-1664
RUN clean-install ca-certificates zlib1g gzip liblzma5 libssl1.1 dpkg
COPY ./init/init-iptables.sh /bin/
RUN chmod +x /bin/init-iptables.sh
# Kubernetes runAsNonRoot requires USER to be numeric
USER 65532:65532

ENTRYPOINT ["./bin/init-iptables.sh"]
