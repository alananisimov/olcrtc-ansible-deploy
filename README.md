# olcrtc deployment

This repository contains the Ansible deployment wrapper and generated
subscription state. The olcrtc source checkout lives in `olcrtc/` as a
separate Git repository, so its upstream changes can be pulled independently.

```sh
make status
make test
make provision
make build
make check
make deploy
```

Local files with infrastructure details or generated keys are ignored:
`ansible/inventory.yml`, `deploy/olcrtc/desired.yaml`, and
`deploy/olcrtc/state.yaml`.

Source repository operations:

```sh
make source-publish       # once: publish the local deploy-tools branch
make source-pull          # update from openlibrecommunity/olcrtc master
```

`make source-pull` always rebases local source commits on
`git@github.com:openlibrecommunity/olcrtc.git master`, regardless of whether
the `deploy-tools` branch has been published to the fork.
