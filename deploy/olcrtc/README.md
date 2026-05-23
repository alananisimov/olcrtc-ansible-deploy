# olcrtc managed subscriptions

The Go source checkout lives in `olcrtc/`; deployment files stay in this
repository. Edit `desired.yaml`, then generate `state.yaml`:

```sh
make provision
```

Build Linux binaries for servers and the subscription backend:

```sh
make build
```

Deploy or preview the Ansible changes:

```sh
make check
make deploy
```

One user gets one URL from `state.yaml` and that URL contains every generated
location/provider entry. `state.yaml` contains room IDs and encryption keys, so
commit or share it only intentionally.
