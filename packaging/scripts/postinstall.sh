#!/bin/sh
set -e

# Create a dedicated system user and group for the service.
if ! getent group gotorrentclient >/dev/null 2>&1; then
    groupadd --system gotorrentclient
fi
if ! getent passwd gotorrentclient >/dev/null 2>&1; then
    useradd --system --gid gotorrentclient \
        --home-dir /var/lib/gotorrentclient --no-create-home \
        --shell /usr/sbin/nologin \
        --comment "GoTorrentClient service" gotorrentclient
fi

# Ensure the state/download directory exists and is owned by the service user.
mkdir -p /var/lib/gotorrentclient/downloads
chown -R gotorrentclient:gotorrentclient /var/lib/gotorrentclient

# Protect credentials in the config file.
chown root:gotorrentclient /etc/gotorrentclient/config.yaml
chmod 640 /etc/gotorrentclient/config.yaml

# Reload systemd so it picks up the unit file.
if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload || true
fi

echo "gotorrentclient installed."
echo "Edit /etc/gotorrentclient/config.yaml, then enable the service:"
echo "  systemctl enable --now gotorrentclient"
