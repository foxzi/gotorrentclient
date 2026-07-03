# deb / rpm packages and the systemd service

gotorrentclient can be installed as a system service from `.deb`
(Debian, Ubuntu, Raspberry Pi OS) or `.rpm` (Fedora, RHEL, Rocky, openSUSE)
packages. The package installs the binary, a systemd unit, a config file under
`/etc`, and creates a dedicated system user.

## What the package installs

| Path | Purpose |
|------|---------|
| `/usr/bin/gotorrentclient` | executable |
| `/lib/systemd/system/gotorrentclient.service` | systemd unit |
| `/etc/gotorrentclient/config.yaml` | config (kept across upgrades) |
| `/var/lib/gotorrentclient/downloads` | downloads and state directory |

The service runs as the `gotorrentclient` user (created on install). The config
file is readable only by root and the service group (mode `0640`).

## Building the packages

Building requires Go and Docker (nfpm runs in a container).

```bash
./packaging/build-packages.sh
```

The script builds static binaries for `amd64` and `arm64`, then uses
[nfpm](https://nfpm.goreleaser.com/) to produce `.deb` and `.rpm` files into
`release/`. The version is derived from `git describe`.

## Installing

Debian / Ubuntu / Raspberry Pi OS:

```bash
sudo dpkg -i gotorrentclient_<version>_arm64.deb
```

Fedora / RHEL / Rocky:

```bash
sudo rpm -i gotorrentclient-<version>.aarch64.rpm
```

Pick the package for your architecture: `amd64` / `x86_64` for regular PCs,
`arm64` / `aarch64` for Raspberry Pi and other ARM boards.

## Running the service

The service is not started automatically on install, so you can configure it
first.

```bash
sudo nano /etc/gotorrentclient/config.yaml   # set username/password
sudo systemctl enable --now gotorrentclient  # enable and start
```

Useful commands:

```bash
systemctl status gotorrentclient      # state
journalctl -u gotorrentclient -f      # live logs
sudo systemctl restart gotorrentclient
```

Once running, the web UI is available on the port from `listen` (default `:8080`).

> Warning: if `username` and `password` are empty, the web UI is open to
> everyone. Set credentials before exposing it to a network.

## Upgrading

Install the new package over the old one — `/etc/gotorrentclient/config.yaml`
is not overwritten (it is marked as a config file). Restart the service after
upgrading:

```bash
sudo systemctl restart gotorrentclient
```

## Uninstalling

```bash
sudo dpkg -r gotorrentclient      # deb
sudo rpm -e gotorrentclient       # rpm
```

The `/var/lib/gotorrentclient` directory (downloads) and the `gotorrentclient`
user are intentionally left in place so data is not lost. For a full cleanup:

```bash
sudo userdel gotorrentclient
sudo rm -rf /var/lib/gotorrentclient
```
