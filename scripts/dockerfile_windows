# This is a Dockerfile that will build a minimal Windows container image with a
# 10 hour sleep command as its entry point. It also adds a secondary user named
# "george" for our synchronization tests.

FROM microsoft/windowsservercore
RUN net user /add george
ENTRYPOINT powershell -Command sleep 6000
