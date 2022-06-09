FROM --platform=${TARGETPLATFORM:-linux/amd64} k8s.gcr.io/build-image/debian-iptables:bullseye-v1.4.0

# upgrading dpkg due to CVE-2022-1664
RUN clean-install ca-certificates dpkg
COPY ./init/init-iptables.sh /bin/
RUN chmod +x /bin/init-iptables.sh
# Kubernetes runAsNonRoot requires USER to be numeric
USER 65532:65532

ENTRYPOINT ["./bin/init-iptables.sh"]
