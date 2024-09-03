#!/bin/env bash

incus ls

incus launch images:debian/bookworm bookworm

incus ls