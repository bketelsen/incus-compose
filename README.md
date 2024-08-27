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
name: cloud  # compose app name, used in custom storage & profile naming
project: default  # incus project to use
export_path: /var/backup/cloud  # where to export backups of instances and volumes (unimplemented)
services:  # list of instances to create
    mydb:
        image: images:ubuntu/noble/cloud
        gpu: true # share the gpu with the instance
        cloud_init_user_data_file: mydb.yaml # if set, creates a custom profile and applies the cloud-init
        additional_profiles: # additional profile to add to this instance
            - nvmestorage
        snapshot: # instance snapshot schedule
            schedule: "@hourly"
            expiry: 2w
        volumes: # incus volumes to attach
            config:
                mountpoint: /config
                pool: fast
        binds: # host directories to bind
            media:
                type: disk
                source: /var/home/bjk
                target: /media
                shift: true
    myjellyfin:
        image: images:ubuntu/noble/cloud
        gpu: true
        cloud_init_user_data_file: jellyfin.yaml
        depends_on: # dependencies, used to determine start/stop order
            - mydb
        snapshot:
            schedule: "@hourly"
            expiry: 2w
        volumes:
            config:
                mountpoint: /config
                pool: default
                snapshot: # volumes aren't included in container snapshots, but you can configure snapshots per volume
                    schedule: "@hourly"
                    expiry: 2w
        binds:
            media:
                type: disk
                source: /var/home/bjk
                target: /media
                shift: true
profiles: # existing profiles to apply to all instances
    - default 
    - vlan5
```