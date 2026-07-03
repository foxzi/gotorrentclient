#!/bin/sh
set -e

# Reload systemd after the unit file has been removed.
if command -v systemctl >/dev/null 2>&1; then
    systemctl daemon-reload >/dev/null 2>&1 || true
fi

# Note: the gotorrentclient system user and /var/lib/gotorrentclient
# (downloaded data) are intentionally left in place so that reinstalling
# does not lose downloads. Remove them manually for a full cleanup:
#   userdel gotorrentclient
#   rm -rf /var/lib/gotorrentclient
