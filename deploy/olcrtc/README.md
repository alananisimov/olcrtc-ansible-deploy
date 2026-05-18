# olcrtc managed subscriptions

Edit `desired.yaml`, then generate `state.yaml`:

```sh
go run ./cmd/olcrtc-provision \
  -config deploy/olcrtc/desired.yaml \
  -state deploy/olcrtc/state.yaml
```

Build Linux binaries for the servers and subscription backend:

```sh
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o build/olcrtc-linux-amd64 ./cmd/olcrtc
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o build/olcrtc-subd-linux-amd64 ./cmd/olcrtc-subd
```

Deploy:

```sh
ansible-playbook -i ansible/inventory.yml ansible/olcrtc.yml
```

One user gets one URL from `state.yaml` and that URL contains every generated
location/provider entry. `state.yaml` contains room IDs and encryption keys, so
commit or share it only intentionally.
