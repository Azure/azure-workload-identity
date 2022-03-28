FROM --platform=${TARGETPLATFORM:-linux/amd64} k8s.gcr.io/build-image/debian-iptables:bullseye-v1.2.0

# upgrading libssl1.1 due to CVE-2021-3711 and CVE-2021-3712
# upgrading libgmp10 due to CVE-2021-43618
# upgrading bsdutils due to CVE-2021-3995 and CVE-2021-3996
# upgrading libc-bin due to CVE-2021-33574, CVE-2022-23218 and CVE-2022-23219
# upgrading libc6 due to CVE-2021-33574, CVE-2022-23218 and CVE-2022-23219
# upgrading libsystemd0 and libudev1 due to CVE-2021-3997
RUN clean-install ca-certificates libssl1.1 libgmp10 bsdutils libc-bin libc6 libsystemd0 libudev1
COPY ./init/init-iptables.sh /bin/
RUN chmod +x /bin/init-iptables.sh
# Kubernetes runAsNonRoot requires USER to be numeric
USER 65532:65532

ENTRYPOINT ["./bin/init-iptables.sh"]
