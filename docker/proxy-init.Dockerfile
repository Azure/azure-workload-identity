FROM --platform=${TARGETPLATFORM:-linux/amd64} registry.k8s.io/build-image/distroless-iptables:v0.4.3

COPY ./init/init-iptables.sh /bin/
RUN chmod +x /bin/init-iptables.sh
# Kubernetes runAsNonRoot requires USER to be numeric
USER 65532:65532

ENTRYPOINT ["./bin/init-iptables.sh"]
