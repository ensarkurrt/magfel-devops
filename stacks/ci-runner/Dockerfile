FROM ubuntu:24.04

ARG RUNNER_VERSION=2.322.0
ARG RUNNER_ARCH=x64

RUN apt-get update && apt-get install -y --no-install-recommends \
    curl ca-certificates git jq docker.io \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -m -s /bin/bash runner \
    && usermod -aG docker runner

WORKDIR /home/runner

RUN curl -sL "https://github.com/actions/runner/releases/download/v${RUNNER_VERSION}/actions-runner-linux-${RUNNER_ARCH}-${RUNNER_VERSION}.tar.gz" \
    | tar xz \
    && ./bin/installdependencies.sh \
    && chown -R runner:runner /home/runner

COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

USER runner

ENTRYPOINT ["/entrypoint.sh"]
