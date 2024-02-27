# OpenZiti Appetizer Demo

This project contains the source for the https://appetizer.openziti.io demo. The demo is currently deployed
into an AWS Fargate environment and has a port exposed so the "underlay" server can listen and deliver
identities for people to use with the appetizer demo.

## Requirements

This project requires a working knowledge of go, basic git knowledge, basic terminal commands

## Running the server

First, start an OpenZiti overlay using the ziti cli quickstart: `ziti edge quickstart` or use whatever
OpenZiti overlay you like.

After ziti is running you can run the underlay and overlay servers using a command like:

#### From Bash
```
OPENZITI_USER="admin" \
OPENZITI_PWD="admin" \
OPENZITI_CTRL="https://localhost:1280" \
OPENZITI_DEMO_INSTANCE="prod" \
go run .\main.go
```
#### From Powershell
```
$env:OPENZITI_USER="admin"
$env:OPENZITI_PWD="admin"
$env:OPENZITI_CTRL="https://localhost:1280"
$env:OPENZITI_DEMO_INSTANCE="prod"
go run .\main.go
```

## Using the Demo

Once Start application locally, you should be able to go to http://localhost:18000/. You'll see a small UI.
Enter your email or some unique id and click the button to "Add to OpenZiti". Read the instructions, 
and click on the link to download token. After you have downloaded token you should be able to `go run` the
examples as shown on the second page.
