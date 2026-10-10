# Libraries the Electron binary links against (even with ELECTRON_RUN_AS_NODE=1).
FROM docker.io/library/debian:bookworm-slim
RUN apt-get update -qq && apt-get install -y -qq --no-install-recommends libnss3 libatk1.0-0 \
    libatk-bridge2.0-0 libcups2 libgtk-3-0 libgbm1 libasound2 libxkbcommon0 libxcomposite1 \
    libxdamage1 libxrandr2 libpango-1.0-0 libcairo2 libdrm2 libxshmfence1 ca-certificates >/dev/null \
    && rm -rf /var/lib/apt/lists/*
