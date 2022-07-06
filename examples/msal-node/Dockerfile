ARG BUILDER=node:14
ARG BASEIMAGE=gcr.io/distroless/nodejs:16

# ref: https://github.com/GoogleContainerTools/distroless/blob/main/examples/nodejs/Dockerfile
FROM ${BUILDER} AS build-env
ADD . /app
WORKDIR /app
RUN npm install

FROM ${BASEIMAGE}
COPY --from=build-env /app /app
WORKDIR /app
# Kubernetes runAsNonRoot requires USER to be numeric
USER 65532:65532
CMD ["index.js"]
