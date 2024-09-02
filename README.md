# incus-compose

`incus-compose` is the missing equivalent for `docker-compose` in the Incus ecosystem.


## Status

Partially functional, but many commands not complete. 

Usage not recommended. 

USE AT YOUR OWN RISK

`incus-compose up` - works

`incus-compose rm` - works

`incus-compose info` - works

`incus-compose snapshot` - works

`incus-compose export` - works



This project currently mixes API & CLI commands, so it only works against a "local" server for now.

## Usage

See the samples in the `samples` directory.

## Explanation

```

x-incus-default-profiles: 
  - default

services:
  drone-server:
    image: images:debian/bookworm/cloud
    container_name: drone-server
    restart: unless-stopped
    x-incus-cloud-init-user-data-file: trixie.yaml
    x-incus-gpu: true
    volumes:
      - ./drone/data:/var/lib/drone
      - myvolume:/mnt/drone/myvolume
      - type: bind
        source: /var/home/bjk/
        target: /mnt/drone/myhome
        x-incus-shift: true
    environment:
      - DRONE_DEBUG=true

      - DRONE_SERVER_PORT=:80
      - DRONE_DATABASE_DRIVER=sqlite3
      - DRONE_GIT_ALWAYS_AUTH=false
      - DRONE_GITEA_SERVER=https://git.domain.tld # change this to your gitea instance
      - DRONE_RPC_SECRET=8aff725d2e16ef31fbc42
      - DRONE_SERVER_HOST=drone.domain.tld # change this to your drone instance
      - DRONE_HOST=https://drone.domain.tld # change this to your drone instance; adjust http/https
      - DRONE_SERVER_PROTO=https # adjust http/https
      - DRONE_TLS_AUTOCERT=false
      - DRONE_AGENTS_ENABLED=true
      - DRONE_GITEA_CLIENT_ID=XXX-XXX # change this to your client ID from Gitea; see https://docs.drone.io/server/provider/gitea/
      - DRONE_GITEA_CLIENT_SECRET=XXX-XXX # change this to your client secret from Gitea; see https://docs.drone.io/server/provider/gitea/
    networks:
      - br0.5 
    depends_on:
      - drone-agent

  drone-agent:
    image: images:debian/bookworm/cloud
    command: agent
    restart: unless-stopped
    container_name: drone-agent
    x-incus-cloud-init-user-data-file: trixie.yaml
    x-incus-snapshot:
      schedule: '@daily'
      expiry: 14d
    environment:
      - DRONE_RPC_SERVER=http://drone-server:80
      - DRONE_RPC_SECRET=8aff725d2e16ef31fbc42
      - DRONE_RUNNER_CAPACITY=2
    networks:
      - br0.5

networks:
  br0.5:
    external: true

volumes:
  myvolume:
    x-incus-snapshot:
      schedule: '@hourly'
      expiry: 2d
    driver: incus
    driver_opts: 
      pool: "default"

```

# TODO

- [ ] environment variables, enumerate & bind
- [ ] networks key, what to do?
- [ ] validation, unsupported keys. warn, abort?
- [ ] ports, when to proxy, POLS