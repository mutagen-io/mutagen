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
replace any `docker-compose` command with `mutagen compose`. For more
information, check out the
[documentation](https://mutagen.io/documentation/orchestration/compose).
