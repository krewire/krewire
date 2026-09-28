FROM nginx:alpine

# Copy custom Nginx configuration
COPY docker/nginx.conf /etc/nginx/conf.d/default.conf

# Copy compiled static site
COPY .krewire/build /usr/share/nginx/html

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://127.0.0.1/ || exit 1

CMD ["nginx", "-g", "daemon off;"]
