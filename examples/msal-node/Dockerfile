ARG BASEIMAGE=mcr.microsoft.com/mirror/gcr/distroless/nodejs-debian11:16

FROM mcr.microsoft.com/cbl-mariner/base/nodejs:16 as build-env
ADD . /app
WORKDIR /app
RUN npm install

FROM ${BASEIMAGE}
COPY --from=build-env /app /app
WORKDIR /app
# Kubernetes runAsNonRoot requires USER to be numeric
USER 65532:65532
CMD ["index.js"]
