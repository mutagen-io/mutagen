# Use the Jupyter Data Science Notebook image as our base.
FROM jupyter/datascience-notebook:latest

# Create a placeholder for the shared volume and set its ownership. The mounted
# volume will inherit the permissions of the placeholder directory. This needs
# to be done in each container where the volume will be mounted, because the
# volume will inherit the first permissions it sees (which, if absent, default
# to root). See https://github.com/moby/moby/issues/2259 for more information.
RUN ["mkdir", "/home/jovyan/data-science"]
RUN ["chown", "jovyan:users", "/home/jovyan/data-science"]

# Change the default working directory to our synchronization target.
WORKDIR /home/jovyan/data-science

# Override the default notebook command to set a password ('mutagen').
CMD ["start-notebook.sh", "--NotebookApp.password=sha1:3e166aeb6462:ec7d7f843690ef8284105bf66797fc8ae270063f"]
