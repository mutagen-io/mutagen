# Use a recent Postgres base image.
FROM postgres:latest

# Copy in the schema initialization script. The Postgres image will pick up on
# the presence of this script and run it when the container is first created.
COPY ["schema.sql", "/docker-entrypoint-initdb.d/schema.sql"]
