ARG BASEIMAGE=k8s.gcr.io/build-image/debian-iptables:bullseye-v1.5.0

FROM --platform=${TARGETPLATFORM:-linux/amd64} ${BASEIMAGE}

# upgrading gpgv due to CVE-2022-34903
# upgrading libgnutls30 due to CVE-2021-4209
# upgrading libtirpc-common due to CVE-2021-46828
# upgrading libtirpc3 due to CVE-2021-46828
RUN clean-install ca-certificates gpgv libgnutls30 libtirpc-common libtirpc3
COPY ./init/init-iptables.sh /bin/
RUN chmod +x /bin/init-iptables.sh
# Kubernetes runAsNonRoot requires USER to be numeric
USER 65532:65532

ENTRYPOINT ["./bin/init-iptables.sh"]
