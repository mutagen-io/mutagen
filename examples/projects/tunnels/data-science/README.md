# Data science

This example is a clone of the [data science example](../../docker/data-science)
that's been modified to use Mutagen's new
[tunnel transport](https://mutagen.io/documentation/transports/tunnels). This
example is kept simple to illustrate how tunnels work, but they can support far
more complex scenarios and work with infrastructure other than Docker (e.g.
Kubernetes clusters).


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

Finally, you'll need to connect Mutagen with the
[mutagen.io API](https://mutagen.io/login) using the `mutagen login` command.
More information about this can be found in the
[tunnel documentation](https://mutagen.io/documentation/transports/tunnels#logging-in)

To use this project, start by creating a tunnel that can be used to communicate
with remote containers:

```
mutagen tunnel create --name=data-science-tunnel > containers/tunnel/tunnel.tunn
```

In this example, we're building the tunnel hosting credentials directly into an
image, but in practice you'd typically store them using a secret management
command like `docker secret create` or `kubectl create secret` and then access
them from inside a container. Tunnels can be re-used indefinitely, so you would
usually create a tunnel to your remote infrastructure just once (as opposed to
every time you started the project).

Once this is set up, you can start the environment using:

```bash
mutagen project start
```

This project uses Mutagen's `beforeCreate` hook (see `mutagen.yml`) to
initialize the Docker Compose services before establishing synchronization and
forwarding.

Once the environment is running, you can access the Jupyter notebook server at
[http://localhost:8888](http://localhost:8888). The password for the notebook
is `mutagen`. For more information on changing the password, please see the
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

If you wish, you can then delete the tunnel you created using:

```
mutagen tunnel terminate data-science-tunnel
```
