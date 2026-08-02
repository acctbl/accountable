# syntax=docker/dockerfile:1.7

FROM ubuntu:24.04

ARG TARGETARCH
ARG MISE_VERSION=2026.7.13

USER root

RUN apt-get update \
	&& apt-get install --yes --no-install-recommends \
		bash \
		ca-certificates \
		curl \
		git \
		sudo \
		tar \
		unzip \
		xz-utils \
		fonts-liberation \
		fonts-noto-color-emoji \
		libasound2t64 \
		libatomic1 \
		libatk-bridge2.0-0t64 \
		libatk1.0-0t64 \
		libatspi2.0-0t64 \
		libcairo2 \
		libcups2t64 \
		libdbus-1-3 \
		libdrm2 \
		libgbm1 \
		libglib2.0-0t64 \
		libnspr4 \
		libnss3 \
		libpango-1.0-0 \
		libx11-6 \
		libxcb1 \
		libxcomposite1 \
		libxdamage1 \
		libxext6 \
		libxfixes3 \
		libxkbcommon0 \
		libxrandr2 \
	&& rm -rf /var/lib/apt/lists/* \
	&& groupadd devcontainer \
	&& useradd --gid devcontainer --create-home --shell /bin/bash devcontainer \
	&& echo 'devcontainer ALL=(root) NOPASSWD:ALL' > /etc/sudoers.d/devcontainer \
	&& chmod 0440 /etc/sudoers.d/devcontainer \
	&& case "${TARGETARCH}" in \
		amd64) mise_arch=x64; mise_sha256=c0d9e71f51bee64e11f1cb9fc179023daed77fe1e594cfb88b615ce4365c2b47 ;; \
		arm64) mise_arch=arm64; mise_sha256=f115d1f911b8eed1bdc9d889d94ff6fdaf892131d573b5678914b2bcf14b2965 ;; \
		*) echo "unsupported architecture: ${TARGETARCH}" >&2; exit 1 ;; \
	esac \
	&& curl --fail --location --silent --show-error \
		"https://github.com/jdx/mise/releases/download/v${MISE_VERSION}/mise-v${MISE_VERSION}-linux-${mise_arch}" \
		--output /tmp/mise \
	&& echo "${mise_sha256}  /tmp/mise" | sha256sum --check \
	&& install --mode=0755 /tmp/mise /usr/local/bin/mise \
	&& rm /tmp/mise

USER devcontainer

ENV MISE_DATA_DIR=/home/devcontainer/.local/share/mise
ENV MISE_CONFIG_DIR=/home/devcontainer/.config/mise
ENV PATH=/home/devcontainer/.local/share/mise/shims:${PATH}
