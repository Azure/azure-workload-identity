FROM --platform=${TARGETPLATFORM:-linux/amd64} registry.k8s.io/build-image/distroless-iptables:v0.10.0@sha256:76ce10a470dd094a79ecd46b85255a0707b4b4dd0603fa8c38b8db4bc154a09a

COPY ./init/init-iptables.sh /bin/
RUN chmod +x /bin/init-iptables.sh
# Kubernetes runAsNonRoot requires USER to be numeric
USER 65532:65532

ENTRYPOINT ["./bin/init-iptables.sh"]
