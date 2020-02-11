# Use the Jupyter Data Science Notebook image as our base.
FROM jupyter/datascience-notebook:latest

# Create a directory that will serve as our synchronization target and the
# working directory for the container.
RUN ["mkdir", "/home/jovyan/data-science"]
RUN ["chown", "jovyan:users", "/home/jovyan/data-science"]
WORKDIR /home/jovyan/data-science

# Override the default notebook command to set a password ('mutagen').
CMD ["start-notebook.sh", "--NotebookApp.password=sha1:3e166aeb6462:ec7d7f843690ef8284105bf66797fc8ae270063f"]
