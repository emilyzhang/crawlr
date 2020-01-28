FROM postgres:12.1-alpine
# pass these in via build arguments instead of hardcoding!
ENV POSTGRES_USER user
ENV POSTGRES_PASSWORD test
ENV POSTGRES_DB crawlr

COPY migrations /docker-entrypoint-initdb.d/
