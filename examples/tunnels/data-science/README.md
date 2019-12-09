# Data science

---

**Note:** This example uses as-of-yet unreleased Mutagen features. These
features will ship with Mutagen v0.11.0. If you're interested in trying them
early, please email [hello@mutagen.io](mailto:hello@mutagen.io).

---

This directory contains an example Python data science environment designed to
run on a cloud-based container host (though it can also be run locally). It uses
Mutagen's tunnel transport to synchronize code from the local editor to the
remote environment and to forward local network traffic to a Jupyter notebook
server running in the remote environment.


## Usage

This section assumes that you have a Docker daemon available and that the
`DOCKER_HOST` environment variable has been set to point to that daemon.

It also assumes that you've logged in to [mutagen.io](https://mutagen.io) using
the `mutagen` command line interface.

To use this project, start by creating a tunnel that can be used to communicate
with remote containers:

```
mutagen tunnel create --name=data-science-tunnel > containers/tunnel/tunnel.tunn
```

Next, start the Mutagen sessions for this project that will communicate over the
tunnel:

```
mutagen project start
```

Finally, start the environment using:

```
docker-compose up --build --detach
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

The remote environment can be terminated using:

```
docker-compose down --rmi=all
```

If you also want to remove the volume created to store synchronized code on the
remote system, you can include the `--volumes` flag when using
`docker-compose down`.

You can start, stop, and rebuild the remote environment as many times as you'd
like, without the need to run any Mutagen commands. If you do remove the remote
volumes, then you'll want to reset the Mutagen synchronization sessions using
`mutagen project reset` before recreating the remote infrastructure.

Once you're finished using the project, you can stop the Mutagen synchronization
and forwarding by using:

```
mutagen project terminate
```

If you wish, you can then delete the tunnel you created using:

```
mutagen tunnel terminate data-science-tunnel
```
