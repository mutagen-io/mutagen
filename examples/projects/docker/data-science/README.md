# Data science

**NOTE:** This example only exists to demonstrate Mutagen's generic
[project](https://mutagen.io/documentation/orchestration/projects) mechanism. In
real-world usage, you'll probably want to use
[Mutagen's support for Docker Compose](https://mutagen.io/documentation/orchestration/compose),
and you can find a corresponding example of this demo
[here](https://github.com/mutagen-io/mutagen/tree/master/examples/compose/data-science).

This directory contains an example [Jupyter](https://jupyter.org/)-based data
science environment designed to be run on a cloud-based container host (though
it can also be run locally, e.g. via
[Docker Desktop](https://www.docker.com/products/docker-desktop)). It uses
[Mutagen's support for Docker containers](https://mutagen.io/documentation/transports/docker)
to synchronize code from the local filesystem to the container filesystem and to
forward network traffic to the containerized Jupyter notebook server.


## Usage

This section assumes that you have a Docker daemon available to the `docker`
command. You can achieve this by running a local Docker daemon with a tool like
[Docker Desktop](https://www.docker.com/products/docker-desktop) or by
configuring access to a cloud-based container host like
[RancherOS](https://rancher.com/rancher-os/) and setting the `DOCKER_HOST`
environment variable appropriately. Mutagen will work with either of these
cases, though setting up a cloud-based container host has numerous performance
benefits.

This section also assumes and that you have
[Docker Compose](https://docs.docker.com/compose/) installed. This is often
bundled with Docker client installations, so check to see if you have it already
by invoking `docker-compose version`.

Once the Docker daemon is set up, you can start the environment using:

```bash
mutagen project start
```

This project uses Mutagen's `beforeCreate` hook (see `mutagen.yml`) to
initialize the Docker Compose services before establishing synchronization and
forwarding.

Once the environment is running, you can access the Jupyter notebook server at
[http://localhost:8888](http://localhost:8888). The password for the notebook
server is `mutagen`. For information on changing the password, please see the
[Jupyter documentation](https://jupyter-docker-stacks.readthedocs.io/en/latest/using/common.html#notebook-options),
as well as the `jupyter` container definition in the `containers` directory.

You can also work inside the containers by starting a shell. For example, try
running `docker-compose exec jupyter bash`. This will start an interactive
shell inside the `jupyter` service container.

To help automate common workflows, Mutagen offers a way to define custom
commands for projects. For example, try running the following:

```bash
mutagen project run ipython
```

This invokes a custom command called `ipython` (defined in `mutagen.yml` using
the `commands` section) that will drop you into an
[IPython](https://ipython.org/) shell running inside the container.

Using custom commands, you can define shells, common analysis workflows, data
processing commands, and more. Defining these common workflows becomes even more
powerful when everyone on a team is using the same environment with the same
tools available.

Once you're done working, you can terminate the environment using:

```bash
mutagen project terminate
```

This project uses Mutagen's `afterTerminate` hook (see `mutagen.yml`) to tear
down the Docker Compose services (and associated resources) after terminating
synchronization and forwarding.
