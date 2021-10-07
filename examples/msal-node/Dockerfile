# ref: https://github.com/GoogleContainerTools/distroless/blob/main/examples/nodejs/Dockerfile

FROM node:14 AS build-env
ADD . /app
WORKDIR /app
RUN npm install

FROM gcr.io/distroless/nodejs:14
COPY --from=build-env /app /app
WORKDIR /app
# Kubernetes runAsNonRoot requires USER to be numeric
USER 65532:65532
CMD ["index.js"]
