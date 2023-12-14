FROM mcr.microsoft.com/mirror/docker/library/debian:bookworm-slim AS build
RUN apt-get update && \
    apt-get install --no-install-suggests --no-install-recommends --yes python3-venv gcc libpython3-dev && \
    python3 -m venv /venv && \
    /venv/bin/pip install --upgrade pip setuptools wheel

# Build the virtualenv as a separate step: Only re-execute this step when requirements.txt changes
FROM build AS build-venv
COPY requirements.txt /requirements.txt
RUN /venv/bin/pip install --disable-pip-version-check -r /requirements.txt

# Copy the virtualenv into a distroless image
FROM mcr.microsoft.com/cbl-mariner/distroless/python:3.9
COPY --from=build-venv /venv /venv
COPY . /app
WORKDIR /app
# Kubernetes runAsNonRoot requires USER to be numeric
USER 65532:65532
ENTRYPOINT ["/venv/bin/python3", "main.py"]
