ARG BUILDER=debian:11-slim
ARG BASEIMAGE=gcr.io/distroless/python3-debian11

# ref: https://github.com/GoogleContainerTools/distroless/blob/main/examples/python3-requirements/Dockerfile
FROM ${BUILDER}  AS build
RUN apt-get update && \
    apt-get install --no-install-suggests --no-install-recommends --yes python3-venv gcc libpython3-dev && \
    python3 -m venv /venv && \
    /venv/bin/pip install --upgrade pip setuptools wheel

# Build the virtualenv as a separate step: Only re-execute this step when requirements.txt changes
FROM build AS build-venv
COPY requirements.txt /requirements.txt
RUN /venv/bin/pip install --disable-pip-version-check -r /requirements.txt

# Copy the virtualenv into a distroless image
FROM ${BASEIMAGE}
COPY --from=build-venv /venv /venv
COPY . /app
WORKDIR /app
# Kubernetes runAsNonRoot requires USER to be numeric
USER 65532:65532
ENTRYPOINT ["/venv/bin/python3", "main.py"]
