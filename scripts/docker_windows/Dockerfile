# Set the base image.
FROM microsoft/windowsservercore

# Add a user.
RUN net user /add george

# Add the HTTP demo server.
COPY ["httpdemo", "C:\\httpdemo"]

# Set the HTTP demo server as the entry point.
ENTRYPOINT ["C:\\httpdemo"]
