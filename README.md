# incus-compose

`incus-compose` is an implementation of the `docker-compose` specification for the Incus ecosystem.


## Status

Partially functional, but many commands not complete. 

Only use this if you know enough `incus` to dig yourself out of the problems this tool might cause.

USE AT YOUR OWN RISK


## Installation

```sh
go install github.com/bketelsen/incus-compose@main
```

## Usage

See the samples in the `samples` directory. Some of them use `x-` attributes to control how `incus-compose` handles a resource.

### docker.io and ghcr.io images

Simply add the remote as `docker.io` or `ghcr.io` to your incus server:

```sh
incus remote add --protocol oci docker.io https://docker.io
incus remote add --protocol oci ghcr.io https://ghcr.io
```

Now you can use `incus-compose` to pull and run images from those remotes, e.g.:

```yaml
services:
  myservice:
    image: docker.io/library/alpine:latest
```
