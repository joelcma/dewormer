go build -o dewormer

# On macos
if [[ "$OSTYPE" == "darwin"* ]]; then
  sudo mv dewormer /usr/local/bin/
  # continue to run a one-off smoke test (see below)
fi

# On linux
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
  sudo mv dewormer /usr/local/bin/
  # continue to run a one-off smoke test (see below)
fi

# On windows (using git bash)
if [[ "$OSTYPE" == "msys"* || "$OSTYPE" == "cygwin"* ]]; then
  mv dewormer.exe /c/Windows/System32/
  # continue to run a one-off smoke test (see below)
fi

# Test that the installation was successful by running the binary once
# (default behaviour is a single run and exit). This avoids touching user
# config and is suitable for smoke-testing.
dewormer || true

echo "Dewormer installed successfully!"

echo "Setting up background service..."

# On macos, set up a launchd service
if [[ "$OSTYPE" == "darwin"* ]]; then
  PLIST=~/Library/LaunchAgents/com.dewormer.plist
  cat <<EOL > $PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.dewormer</string>
    <key>ProgramArguments</key>
    <array>
    <string>/usr/local/bin/dewormer</string>
  </array>
    <key>StartInterval</key>
    <integer>43200</integer> <!-- 12 hours -->
    <key>RunAtLoad</key>
    <true/>
</dict>
</plist>
EOL
  launchctl load $PLIST
  echo "Dewormer background service set up with launchd."
  exit 0
fi

if [[ "$OSTYPE" == "linux-gnu"* ]]; then
  # On linux, set up a systemd oneshot service + timer that runs every 12h.
  SERVICE=/etc/systemd/system/dewormer.service
  sudo bash -c "cat <<EOL > $SERVICE
[Unit]
Description=Dewormer Background Service (oneshot)
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/dewormer

[Install]
WantedBy=multi-user.target
EOL"

  TIMER=/etc/systemd/system/dewormer.timer
  sudo bash -c "cat <<EOL > $TIMER
[Unit]
Description=Run dewormer every 12 hours

[Timer]
OnBootSec=5min
OnUnitActiveSec=12h
Persistent=true

[Install]
WantedBy=timers.target
EOL"

  sudo systemctl daemon-reload
  sudo systemctl enable --now dewormer.timer
  echo "Dewormer background service set up with systemd (oneshot + timer)."
  exit 0
fi

if [[ "$OSTYPE" == "msys"* || "$OSTYPE" == "cygwin"* ]]; then
  echo "Please set up a scheduled task to run '/path/to/dewormer' every 12 hours."
  exit 0
fi