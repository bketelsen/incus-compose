# https://taskfile.dev

version: '3'

tasks:
 
  site:
    desc: Run hugo dev server
    cmds:
      - task: :build
      - task: :generate
      - hugo server --buildDrafts --disableFastRender
  installer:
    desc: Copy installer from root to site/static directory
    cmds:
      - cp ./install.sh ./site/static/install.sh