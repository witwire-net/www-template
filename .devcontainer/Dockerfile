ARG NODE_VERSION=24.12.0
FROM node:${NODE_VERSION}-bookworm

ARG GO_VERSION=1.26.2
ARG PNPM_VERSION=11.1.1
ARG TYPESPEC_VERSION=1.8.0
ARG GORM_VERSION=1.31.0
ARG GOLANG_MIGRATE_VERSION=4.18.3
ARG OAPI_CODEGEN_VERSION=2.4.1
ARG GOLANGCI_LINT_VERSION=1.64.8
ARG WRANGLER_VERSION=4.57.0
ARG OPENSPEC_VERSION=1.2.0
ARG OPENCODE_VERSION=1.14.48

ENV DEBIAN_FRONTEND=noninteractive
ENV NPM_CONFIG_PREFIX=/home/node/.npm-global
ENV PNPM_HOME=/home/node/.local/share/pnpm
ENV GOPATH=/go
ENV GOMODCACHE=/go/pkg/mod
ENV GOCACHE=/home/node/.cache/go-build
ENV PLAYWRIGHT_BROWSERS_PATH=/ms-playwright
ENV PATH=/usr/local/go/bin:/home/node/.local/bin:/home/node/.npm-global/bin:/home/node/.local/share/pnpm:/go/bin:$PATH

RUN apt-get update && apt-get install -y \
    build-essential \
    ca-certificates \
    curl \
    git \
    jq \
    make \
    pkg-config \
    postgresql-client \
    python3 \
    python3-pip \
    python3-venv \
    redis-tools \
    unzip \
    xz-utils \
    libnspr4 \
    libnss3 \
    libdbus-1-3 \
    libatk1.0-0 \
    libatspi2.0-0 \
    libxcomposite1 \
    libxdamage1 \
    libxfixes3 \
    libxrandr2 \
    libgbm1 \
    libxkbcommon0 \
    libasound2 \
    libx11-xcb1 \
    libxcursor1 \
    libgtk-3-0 \
    libgtk-4-1 \
    libgraphene-1.0-0 \
    libwoff1 \
    libvpx7 \
    libopus0 \
    libgstreamer1.0-0 \
    libgstreamer-gl1.0-0 \
    gstreamer1.0-libav \
    gstreamer1.0-plugins-base \
    gstreamer1.0-plugins-good \
    gstreamer1.0-plugins-bad \
    flite \
    libavif15 \
    libharfbuzz-icu0 \
    libepoxy0 \
    libenchant-2-2 \
    libsecret-1-0 \
    libhyphen0 \
    libwayland-egl1 \
    libwayland-client0 \
    libmanette-0.2-0 \
    libatk-bridge2.0-0 \
    libgles2 \
    libx264-164 \
    && rm -rf /var/lib/apt/lists/*

RUN arch="$(dpkg --print-architecture)" \
    && case "$arch" in \
      amd64) go_arch='amd64' ;; \
      arm64) go_arch='arm64' ;; \
      *) echo "Unsupported architecture: $arch" && exit 1 ;; \
    esac \
    && curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${go_arch}.tar.gz" -o /tmp/go.tgz \
    && rm -rf /usr/local/go \
    && tar -C /usr/local -xzf /tmp/go.tgz \
    && rm /tmp/go.tgz

RUN curl -LsSf https://astral.sh/uv/install.sh | sh

RUN chmod +x /root/.local/bin/uv /root/.local/bin/uvx \
    && cp /root/.local/bin/uv /usr/local/bin/uv \
    && cp /root/.local/bin/uvx /usr/local/bin/uvx \
    && chmod 755 /usr/local/bin/uv /usr/local/bin/uvx

RUN corepack enable

RUN mkdir -p \
    /go/pkg/mod \
    /ms-playwright \
    /home/node/.cache/go-build \
    /home/node/.local/bin \
    /home/node/.local/share/pnpm \
    /home/node/.local/share/pnpm/store \
    /home/node/.npm-global \
    && chown -R node:node \
    /go \
    /ms-playwright \
    /home/node/.cache \
    /home/node/.local \
    /home/node/.npm-global \
    && printf '%s\n' \
    'export NPM_CONFIG_PREFIX="/home/node/.npm-global"' \
    'export PNPM_HOME="/home/node/.local/share/pnpm"' \
    'export GOPATH="/go"' \
    'export GOMODCACHE="/go/pkg/mod"' \
    'export GOCACHE="/home/node/.cache/go-build"' \
    'export PLAYWRIGHT_BROWSERS_PATH="/ms-playwright"' \
    'export PATH="/usr/local/go/bin:/home/node/.local/bin:/home/node/.npm-global/bin:/home/node/.local/share/pnpm:/go/bin:$PATH"' \
    > /etc/profile.d/99-template-paths.sh

USER node

RUN corepack prepare pnpm@${PNPM_VERSION} --activate

ENV GOBIN=/home/node/.local/bin

RUN npm install -g "wrangler@${WRANGLER_VERSION}" "@typespec/compiler@${TYPESPEC_VERSION}" "@fission-ai/openspec@${OPENSPEC_VERSION}" "opencode-ai@${OPENCODE_VERSION}"

RUN go install golang.org/x/tools/gopls@latest \
    && go install golang.org/x/tools/cmd/goimports@latest \
    && go install github.com/go-delve/delve/cmd/dlv@latest \
    && go install github.com/golangci/golangci-lint/cmd/golangci-lint@v${GOLANGCI_LINT_VERSION} \
    && go install github.com/golang-migrate/migrate/v4/cmd/migrate@v${GOLANG_MIGRATE_VERSION} \
    && go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@v${OAPI_CODEGEN_VERSION} \
    && go install github.com/air-verse/air@latest
