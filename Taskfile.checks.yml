# https://taskfile.dev
# Generated File, changes may be lost
# Add `Taskfile.custom.yml` in this directory with your additions

version: '3'

tasks:
  test:
    desc: Run all tests
    cmds:
      - go test ./...
    silent: true

  format:
    desc: Format all Go source
    cmds:
      - gofmt -w -s .
    silent: true

  vet:
    desc: Run go vet on sources   
    cmds:
      - go vet ./...
    silent: true

  staticcheck:
    desc: Run go staticcheck
    cmds:
      - staticcheck ./...
    silent: true

  tidy:
    desc: Run go mod tidy 
    cmds:
      - go mod tidy
    silent: true

  all:
    desc: Run all go checks
    deps: [tidy,format, staticcheck, vet, test]
    silent: true
