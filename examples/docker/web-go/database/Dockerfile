# Use a recent Postgres base image.
FROM postgres:latest

# Set environment variables to control database parameters. The parameters here
# are for demonstration and development purposes only. For production systems,
# you should use a secret management mechanism.
ENV POSTGRES_DB="messagesdb" \
    POSTGRES_USER="messagessvc" \
    POSTGRES_PASSWORD='InSeCUr3Dem0PasS'

# Copy in the schema initialization script. The Postgres image will pick up on
# the presence of this script and run it when the container is first created.
COPY ["schema.sql", "/docker-entrypoint-initdb.d/schema.sql"]
