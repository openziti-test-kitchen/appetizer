# OpenZiti Appetizer Demo

This project contains the source for the https://appetizer.openziti.io demo. The demo is currently deployed
into an AWS Fargate environment and has a port exposed so the "underlay" server can listen and deliver
identities for people to use with the appetizer demo.

## Requirements

This project requires a working knowledge of go, basic git knowledge, basic terminal commands

## Environment Variables

| Variable | Required | Default | Description |
|---|---|---|---|
| `OPENZITI_CTRL` | yes | | URL of the OpenZiti controller (e.g. `https://localhost:1280`) |
| `OPENZITI_USER` | yes | | Admin username for the controller |
| `OPENZITI_PWD` | yes | | Admin password for the controller |
| `OPENZITI_DEMO_INSTANCE` | no | hostname | Instance name used to namespace services. Set to `prod` to use unprefixed service names. |
| `OPENZITI_RECREATE_NETWORK` | no | `true` | When `true`, recreates the OpenZiti network config on startup. Set to `false` to skip recreation and reuse existing config. |

## Running the server locally

First, start an OpenZiti overlay using the ziti cli quickstart: `ziti edge quickstart` or use whatever
OpenZiti overlay you like.

After ziti is running you can run the underlay and overlay servers using a command like:

#### From Bash
```bash
OPENZITI_USER="admin" \
OPENZITI_PWD="admin" \
OPENZITI_CTRL="https://localhost:1280" \
OPENZITI_DEMO_INSTANCE="prod" \
go run ./main.go
```
#### From Powershell
```powershell
$env:OPENZITI_USER="admin"
$env:OPENZITI_PWD="admin"
$env:OPENZITI_CTRL="https://localhost:1280"
$env:OPENZITI_DEMO_INSTANCE="prod"
go run .\main.go
```

## Running with Docker Compose

The `docker-compose.yml` file spins up a full local stack: a ziti quickstart controller and four appetizer
instances (default, staging, prod, local) each on a different port.

```bash
docker compose up
```

| Instance | Port |
|---|---|
| default (hostname) | http://localhost:18004 |
| staging | http://localhost:18001 |
| prod | http://localhost:18002 |
| local | http://localhost:18003 |

## Building and Publishing the Container

`publishContainer.sh` builds the Go binary, embeds the current git SHA into `http_content/version.html`,
and builds a multi-arch Docker image (`linux/amd64` and `linux/arm64`).

**Push to Docker Hub** (tags as `openziti/appetizer:latest`):
```bash
./publishContainer.sh
```

**Load locally** (for testing with Docker Compose before pushing):
```bash
./publishContainer.sh local
```

The script requires `docker buildx`. The image is published to `openziti/appetizer:latest` on Docker Hub.

## Using the Demo

Once the application is running, go to http://localhost:18000/. You'll see a small UI.
Enter your email or some unique id and click the button to "Add to OpenZiti". Read the instructions,
and click on the link to download token. After you have downloaded token you should be able to `go run` the
examples as shown on the second page.
