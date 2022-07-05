ARG BUILDER=maven:3.8.4-jdk-11
ARG BASEIMAGE=gcr.io/distroless/java:11-nonroot

FROM ${BUILDER} as builder
WORKDIR /app
COPY pom.xml .
RUN mvn -e -B dependency:resolve
COPY src ./src
RUN mvn -e -B package

FROM ${BASEIMAGE}
COPY --from=builder /app/target/msal-java-*.jar /app.jar
# Kubernetes runAsNonRoot requires USER to be numeric
USER 65532:65532
CMD ["/app.jar"]
