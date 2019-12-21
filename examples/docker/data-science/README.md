# Data science

This directory contains an example Python data science environment designed to
run on a cloud-based container host (though it can also be run locally). It uses
[Mutagen's support for Docker containers](https://mutagen.io/documentation/transports/docker)
to synchronize code from the local editor to the remote environment and to
forward local network traffic to a Jupyter notebook server running in the remote
environment.


## Usage

This section assumes that you have a Docker daemon available and that the
`DOCKER_HOST` environment variable has been set to point to that daemon.

First, start the environment using:

```
docker-compose up --build --detach
```

Next, start the Mutagen synchronization and forwarding sessions for this project
that will communicate with the containers:

```
mutagen project start
```

Once the environment is running, you can access the Jupyter notebook server at
[http://localhost:8888](http://localhost:8888). The password for the notebook
is `mutagen`. For more information on changing the password, please see the
[Jupyter documentation](https://jupyter-docker-stacks.readthedocs.io/en/latest/using/common.html#notebook-options),
as well as the `jupyter` container definition in the `containers` directory.

You can also create interactive command-line sessions in the remote environment,
e.g.:

```
docker-compose exec jupyter bash
```

or

```
docker-compose exec jupyter ipython
```

Once you're done working with the remote environment, you can terminate the
Mutagen sessions using:

```
mutagen project terminate
```

The remote environment can be terminated using:

```
docker-compose down --rmi=all
```

If you also want to remove the volume created to store synchronized code on the
remote system, you can include the `--volumes` flag when using
`docker-compose down`.
