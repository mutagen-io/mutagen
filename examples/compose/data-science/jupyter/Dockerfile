# Use the Jupyter Data Science Notebook image as our base.
FROM jupyter/datascience-notebook:latest

# Set the working directory to the location where we'll synchronize code.
WORKDIR /code/data-science

# Override the default notebook command to set a password ('mutagen').
CMD ["start-notebook.sh", "--NotebookApp.password=sha1:3e166aeb6462:ec7d7f843690ef8284105bf66797fc8ae270063f"]
