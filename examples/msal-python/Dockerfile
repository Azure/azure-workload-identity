# ref: https://github.com/GoogleContainerTools/distroless/blob/main/examples/python3-requirements/Dockerfile

FROM debian:buster-slim AS build
RUN apt-get update && \
    apt-get install --no-install-suggests --no-install-recommends --yes python3-venv gcc libpython3-dev && \
    python3 -m venv /venv && \
    /venv/bin/pip install --upgrade pip

# Build the virtualenv as a separate step: Only re-execute this step when requirements.txt changes
FROM build AS build-venv
COPY requirements.txt /requirements.txt
RUN /venv/bin/pip install --disable-pip-version-check -r /requirements.txt

# Copy the virtualenv into a distroless image
FROM gcr.io/distroless/python3-debian11
COPY --from=build-venv /venv /venv
COPY . /app
WORKDIR /app
# Kubernetes runAsNonRoot requires USER to be numeric
USER 65532:65532
ENTRYPOINT ["/venv/bin/python3", "main.py"]
