#!/bin/env bash

incus file push init.yaml composetest/root/init.yaml

incus exec composetest -- bash "incus admin init  < /root/init.yaml"

incus file push ../incus-compose composetest/usr/local/bin/incus-compose