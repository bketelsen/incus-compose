# https://taskfile.dev
# Generated File, changes may be lost
# Add `Taskfile.custom.yml` in this directory with your additions

version: '3'

tasks:
  snapshot:
    desc: Run goreleaser in snapshot mode
    cmds:
      - goreleaser release --snapshot --clean
    silent: true

  release-check:
    desc: Run goreleaser check
    cmds:
      - goreleaser check
    silent: true

  publish:
    desc: Push and tag at {{.VERSION}}
    cmds:
      - git push origin
      - git tag v{{.VERSION}}
      - git push --tags

  goreleaser:
    desc: Install goreleaser on debian derivatives
    cmds:
      - wget https://github.com/goreleaser/goreleaser-pro/releases/download/v2.7.0-pro/goreleaser-pro_2.7.0_amd64.deb
      - sudo dpkg -i goreleaser-pro_2.7.0_amd64.deb
      - rm goreleaser-pro_2.7.0_amd64.deb
    silent: true