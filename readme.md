<div align="center">

<img src="https://github.com/openlibrecommunity/material/blob/master/olcrtc.png" width="180" height="180">

# olcRTC

WebRTC-based tunneling over public conferencing services.

![License](https://img.shields.io/badge/license-WTFPL-0D1117?style=flat-square&logo=open-source-initiative&logoColor=green&labelColor=0D1117)
![Golang](https://img.shields.io/badge/-Golang-0D1117?style=flat-square&logo=go&logoColor=00A7D0)

</div>

## Status

Beta. Expect rough edges.

- Issues: [openlibrecommunity/olcrtc/issues](https://github.com/openlibrecommunity/olcrtc/issues)
- Community UI client: [alananisimov/olcbox](https://github.com/alananisimov/olcbox)
- Contact: [@openlibrecommunity](https://t.me/openlibrecommunity)

## Documentation

- [Quick start](docs/fast.md)
- [Manual setup](docs/manual.md)
- [Configuration](docs/configuration.md)
- [Transport matrix](docs/settings.md)
- [Client URI format](docs/uri.md)
- [Subscription format](docs/sub.md)
- [Project notes](docs/about.md)

## Build

Install Mage first:

```bash
go install github.com/magefile/mage@latest
```

Common targets:

```bash
mage build          # olcrtc, olcrtc-subd, olcrtc-provision
mage buildCLI       # olcrtc only
mage buildSubd      # subscription backend
mage buildProvision # provisioning generator
mage cross          # olcrtc for supported OS/ARCH targets
mage docker         # Docker image
mage podman         # Podman image
mage test
mage lint
mage clean
```

## Ansible Deployment

This repository includes an Ansible playbook for managed server deployments.

Files:

- `ansible/olcrtc.yml` - main playbook
- `ansible/inventory-example.yml` - inventory template
- `deploy/olcrtc/desired-example.yaml` - desired subscription layout template
- `deploy/olcrtc/state.yaml` - generated state consumed by Ansible and `olcrtc-subd`

`state.yaml` contains subscription tokens, room IDs, and encryption keys. Treat it as a secret.

### Requirements

Control machine:

- Go
- Mage
- Ansible
- SSH access to target hosts

Server hosts in `olcrtc_srv`:

- Linux with systemd
- root or sudo access through Ansible `become`

Subscription host in `olcrtc_sub`:

- Docker with `docker compose`
- external Docker network named `traefik`
- Traefik entrypoint `web-secure`
- Traefik cert resolver `default`
- HAProxy config at `/etc/haproxy/haproxy.cfg`

Adjust `ansible/templates/subd-compose.yml.j2` and the HAProxy task in `ansible/olcrtc.yml` if your edge stack is different.

### Deploy

Create and edit the desired state:

```bash
cp deploy/olcrtc/desired-example.yaml deploy/olcrtc/desired.yaml
```

Generate deployment state and subscription URLs:

```bash
go run ./cmd/olcrtc-provision \
  -config deploy/olcrtc/desired.yaml \
  -state deploy/olcrtc/state.yaml
```

Build Linux binaries expected by the playbook:

```bash
GOOS=linux GOARCH=amd64 mage buildCLI buildSubd
```

Create and edit inventory:

```bash
cp ansible/inventory-example.yml deploy/olcrtc/inventory.yml
```

Each `olcrtc_srv` host must set `olcrtc_location_id` to a matching `locations[].id` from `desired.yaml`.

Check SSH access:

```bash
ansible -i deploy/olcrtc/inventory.yml all -m ping
```

Run the playbook:

```bash
ansible-playbook -i deploy/olcrtc/inventory.yml ansible/olcrtc.yml
```

Useful limits:

```bash
ansible-playbook -i deploy/olcrtc/inventory.yml ansible/olcrtc.yml --limit olcrtc_srv
ansible-playbook -i deploy/olcrtc/inventory.yml ansible/olcrtc.yml --limit olcrtc_sub
```

For non-amd64 hosts, build the matching binaries and override the local paths:

```bash
GOOS=linux GOARCH=arm64 mage buildCLI buildSubd

ansible-playbook -i deploy/olcrtc/inventory.yml ansible/olcrtc.yml \
  -e "olcrtc_bin_local=$PWD/build/olcrtc-linux-arm64" \
  -e "olcrtc_subd_bin_local=$PWD/build/olcrtc-subd-linux-arm64"
```

### Verify

On an `olcrtc_srv` host:

```bash
systemctl status 'olcrtc-srv@*'
journalctl -u 'olcrtc-srv@*' -f
```

On the `olcrtc_sub` host:

```bash
docker compose -f /opt/olcrtc-subd/compose.yml ps
curl -fsS https://olcrtc.example.com/sub/<token>
```

## License

[WTFPL](LICENSE)

