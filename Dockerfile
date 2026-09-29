# ── Stage: serve ──────────────────────────────────────────────────────────────
# Pin to digest to prevent unexpected image mutation on re-pull.
# To update: run `docker pull nginx:alpine` then update the digest below.
FROM nginx:alpine@sha256:df221db836e1754089190208cee7eeda94f233197056426eda74a43ab1abeac2

LABEL org.opencontainers.image.title="krewire-web" \
      org.opencontainers.image.description="Krewire static documentation and landing site" \
      org.opencontainers.image.source="https://github.com/krewire/krewire" \
      org.opencontainers.image.version="v0.1.0" \
      org.opencontainers.image.licenses="MIT"

# Remove default nginx config to avoid any default server leakage
RUN rm /etc/nginx/conf.d/default.conf

# Inject hardened nginx configuration
COPY docker/nginx.conf /etc/nginx/conf.d/krewire.conf

# Copy compiled static site artifacts
COPY .krewire/build /usr/share/nginx/html

EXPOSE 80

# Liveness check: nginx process is running and can serve the root page
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://127.0.0.1/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
