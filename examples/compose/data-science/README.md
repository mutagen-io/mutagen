# Data science

This directory contains an example [Jupyter](https://jupyter.org/)-based data
science environment designed to be run on a cloud-based container host (though
it can also be run locally, e.g. via
[Docker Desktop](https://www.docker.com/products/docker-desktop)). It uses
[Mutagen's support for Docker Compose](https://mutagen.io/documentation/orchestration/compose)
to synchronize code from the local filesystem to a shared container volume and
to forward network traffic to the containerized Jupyter notebook server,
allowing you to edit code and access the environment locally, regardless of
where the project is running.


## Usage

This example behaves like any other Composed-based project—you'll just need to
replace any `docker-compose` command with `mutagen compose`. Once the project is
running, you can access the Jupyter notebook server that it creates at
[http://localhost:8888](http://localhost:8888). The password for the notebook
server is `mutagen`. Once the notebook server is running, try making changes to
the analysis code and then reloading it in the remote notebook. For more
information, check out the
[documentation](https://mutagen.io/documentation/orchestration/compose).
