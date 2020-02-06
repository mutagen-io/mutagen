# Set the base image.
FROM microsoft/windowsservercore:1809

# Add a user.
RUN net user /add george

# Add the HTTP demo server.
COPY httpdemo.exe c:/

# Set the HTTP demo server as the entry point.
ENTRYPOINT c:/httpdemo.exe
