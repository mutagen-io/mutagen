# Set the base image.
FROM mcr.microsoft.com/windows/servercore:ltsc2022

# Add a user.
RUN net user /add george

# Add the HTTP demo server.
COPY httpdemo.exe c:/

# Set the HTTP demo server as the entry point.
ENTRYPOINT c:/httpdemo.exe
