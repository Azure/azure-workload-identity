FROM --platform=${TARGETPLATFORM:-linux/amd64} k8s.gcr.io/build-image/debian-iptables:bullseye-v1.1.0

# upgrading libssl1.1 due to CVE-2021-3711 and CVE-2021-3712
# upgrading libgssapi-krb5-2 and libk5crypto3 due to CVE-2021-37750
# upgrading libgmp10 due to CVE-2021-43618
RUN clean-install ca-certificates libssl1.1 libgssapi-krb5-2 libk5crypto3 libgmp10
COPY ./init/init-iptables.sh /bin/
RUN chmod +x /bin/init-iptables.sh
# Kubernetes runAsNonRoot requires USER to be numeric
USER 65532:65532

ENTRYPOINT ["./bin/init-iptables.sh"]
