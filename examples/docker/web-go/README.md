# Go-based web application

This directory contains an example [Go](https://golang.org/)-based web
application designed to be developed and run on a cloud-based container host
(though it can also be run locally, e.g. via
[Docker Desktop](https://www.docker.com/products/docker-desktop)). It uses
[Mutagen's support for Docker containers](https://mutagen.io/documentation/transports/docker)
to synchronize code from the local filesystem to a shared container volume and
to forward network traffic to the various containerized services.


## Design

This application (a web-based message board) is kept intentionally simple, but
still contains multiple services running in separate containers to illustrate
how a more complex application with the same architecture might be organized.
The container orchestration setup outlined here is for development, though a
production setup would look similar (and would typically live inside the same
codebase with slightly different container definitions and orchestration files).

The development setup consists of five services:

- A database service running [PostgreSQL](https://www.postgresql.org/)
- A Go-based API server that handles message read/write requests
- A front-end build service that uses [gulp](https://gulpjs.com/) to build the
  HTML, CSS, and JavaScript that constitute the front-end of the application
- A Go-based web server that serves the front-end content
- A development service that populates a shared volume with an initial copy of
  the code and then waits to host Mutagen agents for code synchronization and
  network forwarding

In a production setup, you would typically only have three services: the
database, the API server, and the web server. The front-end would usually be
statically built and bundled with the web server container image, while the
development service would be unnecessary.

The application itself consists of a simple front-end that allows users to
submit messages (which are sent via AJAX to the API and stored in the database)
and read recently submitted messages (which are loaded from the database and
served by the API via AJAX). It's worth noting that this application is not
production-ready (it has no authentication, CSRF-prevention, rate-limiting,
etc.), but its architecture does mirror that of a fully fledged application and
thus the orchestration setup (both in terms of Docker Compose and Mutagen)
should look very similar to a "real" application.


## Usage

This section assumes that you have a Docker daemon available to the `docker`
command. You can achieve this by running a local Docker daemon with a tool like
[Docker Desktop](https://www.docker.com/products/docker-desktop) or by
configuring access to a cloud-based container host like
[CoreOS](http://coreos.com/) and setting the `DOCKER_HOST` environment variable
appropriately. Mutagen will work with either of these cases, though setting up a
cloud-based container host has numerous performance benefits.

This section also assumes and that you have
[Docker Compose](https://docs.docker.com/compose/) installed. This is often
bundled with Docker client installations, so check to see if you have it already
by invoking `docker-compose version`.

Once the Docker daemon is set up, you can start the project using:

```bash
mutagen project start
```

This project uses Mutagen's project `setup` hook (defined in `mutagen.yml`) to
initialize the Docker Compose containers before establishing synchronization and
forwarding.

Once the environment is running, you can access the application at
[http://localhost:8080](http://localhost:8080).

Try making an edit to the `frontend/index.html` file in your local editor. Once
saved, you can refresh the page to see your changes. It's worth noting that
Mutagen's synchronization will also work with build tools and servers that
perform hot reloads, you'd just need to use one of those tools as the container
entry point instead of the simple setup used here.

The API and web services are configured to rebuild their server binaries on
restart, so if you make a change to the server code, then you'll need to restart
the corresponding service using `docker-compose restart api` or
`docker-compose restart web`. These rebuilds could also be automated using a
tool like [Watchman](https://facebook.github.io/watchman/), but that's outside
the scope of this demo.

You can also work inside the containers by starting a shell. For example, try
running `docker-compose exec frontend sh`. This will start an interactive shell
inside the `frontend` service container.

To help automate common workflows, Mutagen offers a way to define custom
commands for projects. For example, try running the following:

```bash
mutagen project run database
```

This invokes a custom command called `database` (defined in `mutagen.yml` using
the `commands` section). It will drop you into a `psql` shell running in the
database container and connected to the message database. Try running
`SELECT * FROM messages;` after creating some messages in the web application.

Using custom commands, you can define shells, common debugging workflows, build
commands, or even script deployment. Defining these common workflows becomes
even more powerful when every developer on a team is using the same environment
with the same tools available.

Once you're done working, you can terminate the project using:

```bash
mutagen project terminate
```

This project uses Mutagen's `teardown` hook (defined in `mutagen.yml`) to
destroy the Docker Compose containers (and associated resources) after
terminating synchronization and forwarding.


## Synchronization setup

This project uses a volume that shares application code across containers.
Because the containers need access to this code at startup, the `development`
service snapshots the codebase into its image and then copies that snapshot to
the shared volume the first time it runs. The other services wait for the
`development` service to populate the snapshot before reaching their actual
entry points (otherwise they wouldn't be able to start). Once Mutagen is
started, it synchronizes the real working tree with this snapshot and propagates
changes bidirectionally.
