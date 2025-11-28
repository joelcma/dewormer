# Dewormer

A cross-platform background scanner for detecting compromised npm and Maven dependencies in your projects.

## What It Does

Dewormer continuously monitors your development directories for known malicious packages from supply chain attacks. It scans `package-lock.json` and `pom.xml` files, compares dependencies against curated lists of compromised packages, and alerts you via desktop notifications when threats are detected.

**Install it. Configure it. Forget it.**

## Features

- 🔍 **Automatic scanning** - Can run on-demand (single-run) or periodically. The CLI performs a single run when no interval is specified. If you want periodic operation on the command line, invoke the program with the `--interval` flag; for installed services use your platform scheduler (systemd timer / launchd StartInterval / Windows scheduled task).
- 🔔 **Desktop notifications** - Get alerted immediately when threats are found
- 📦 **Multi-ecosystem support** - Scans both npm (package-lock.json) and Maven (pom.xml)
- 🎯 **Customizable** - Configure scan paths and maintain your own bad package lists
- 🪶 **Lightweight** - Single binary, minimal resource usage
- 🖥️ **Cross-platform** - Works on Windows, macOS, and Linux

## Command line

Dewormer exposes a couple of convenient flags:

- `--version` or `-v` — prints the build version and exits.
- `--interval <duration>` or `-i <duration>` — run the program periodically with the supplied duration (e.g. `12h`, `30m`, `24h`). If omitted the program performs a single-run and exits. For production installs prefer scheduling the program to run at intervals using your system's scheduler (systemd timer / launchd StartInterval / Windows scheduled task) instead of relying on `--interval` in a background service.
 - `--config <path>` — path to config.json to use instead of the default `~/.dewormer/config.json`.

## Installation

### Prerequisites

- Go 1.16 or later

### Build from source

```bash
# Clone the repository
git clone https://github.com/joelcma/dewormer.git
cd dewormer

# Install dependencies
go get github.com/gen2brain/beeep

# Build
go build -o dewormer

# Move to your PATH (optional)
sudo mv dewormer /usr/local/bin/
```

### First Run

Default config locations:

- macOS / Linux: `~/.dewormer/config.json`
- Windows: `%USERPROFILE%\\.dewormer\\config.json` (or `$HOME/.dewormer/config.json` in POSIX shells)

```bash
./dewormer
```

On first run, Dewormer will:

1. Create `~/.dewormer/config.json` with default settings
2. Create `~/.dewormer/bad_package_lists/` directory
3. Create an example bad package list with some known malicious packages
4. Exit with instructions to edit the config

## Configuration

Edit `~/.dewormer/config.json`:

```json
{
  "scan_paths": [
    "/Users/yourname/projects",
    "/Users/yourname/work",
    "C:\\Users\\yourname\\projects"
  ]
}
```

**Configuration options:**

- `scan_paths` - List of directories to scan recursively for dependency files

Dewormer will also look for bad package lists in `~/.dewormer/bad_package_lists/`. You can add your own lists or download community-maintained ones.
The lists should be simple text files with one package per line in the format `package-name@version`.

### Persistent scan state

Dewormer keeps a small state file at `~/.dewormer/scan_state.json` which records the last time each scanned dependency file was processed. This allows Dewormer to skip files that haven't changed since the last scan and to only re-scan when either the dependency file changes or any bad package list file has been updated.

## Bad Package Lists

Bad package lists are simple text files with one package per line in the format:

```
package-name@version
```

Example (`~/.dewormer/bad_package_lists/npm-malicious.txt`):

```
# NPM malicious packages - Updated 2024-11-28
# Lines starting with # are comments

voip-callkit@1.0.2
voip-callkit@1.0.3
eslint-config-teselagen@6.1.7
@rxap/ngx-bootstrap@19.0.3
@rxap/ngx-bootstrap@19.0.4
wdio-web-reporter@0.1.3
yargs-help-output@5.0.3
```

For Maven packages, use the format `groupId:artifactId@version`:

```
# Maven compromised packages
com.example:malicious-lib@1.2.3
org.badactor:evil-dependency@2.0.1
```

### Maintaining Bad Package Lists

You can maintain multiple lists and update them independently:

```bash
# Add a new list
echo "bad-package@1.0.0" >> ~/.dewormer/bad_package_lists/custom-list.txt

# Update config to include the new list
nano ~/.dewormer/config.json

# Download community-maintained lists
curl -o ~/.dewormer/bad_package_lists/community-npm.txt \
  https://example.com/bad-packages/npm.txt
```

## Running Dewormer

### Foreground (for testing)

```bash
./dewormer
```

### Background

**macOS/Linux:**

```bash
nohup ./dewormer > ~/.dewormer/dewormer.log 2>&1 &
```

**Windows:**

```cmd
start /B dewormer.exe
```

### As a System Service

#### macOS (launchd)

Create `~/Library/LaunchAgents/com.dewormer.agent.plist`:

```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.dewormer.agent</string>
    <key>ProgramArguments</key>
    <array>
      <string>/usr/local/bin/dewormer</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>/Users/yourname/.dewormer/dewormer.log</string>
    <key>StandardErrorPath</key>
    <string>/Users/yourname/.dewormer/dewormer.log</string>
</dict>
</plist>
```

```bash
launchctl load ~/Library/LaunchAgents/com.dewormer.agent.plist
```

#### Linux (systemd)

Create a short, oneshot service and a timer to run it every 12 hours. Put both files under `/etc/systemd/system`.

`/etc/systemd/system/dewormer.service` (oneshot):

```ini
[Unit]
Description=Dewormer Background Service (oneshot)
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/dewormer

[Install]
WantedBy=multi-user.target
```

`/etc/systemd/system/dewormer.timer` (runs the oneshot every 12 hours):

```ini
[Unit]
Description=Run dewormer every 12 hours

[Timer]
OnBootSec=5min
OnUnitActiveSec=12h
Persistent=true

[Install]
WantedBy=timers.target
```

Enable and start the timer (the timer will invoke the oneshot service on schedule):

```bash
sudo systemctl daemon-reload
sudo systemctl enable --now dewormer.timer
sudo systemctl status dewormer.timer
```

#### Windows (Task Scheduler)

1. Open Task Scheduler
2. Create Basic Task → Name it "Dewormer"
3. Trigger: "When I log on"
4. Action: "Start a program" → Browse to and select the `dewormer` binary
5. Check "Run whether user is logged on or not"

## How It Works

1. **File Discovery** - Recursively scans configured directories for `package-lock.json` and `pom.xml` files
2. **Dependency Extraction** - Parses JSON/XML and extracts all dependencies with versions
3. **Normalization** - Converts to standardized `package@version` format
4. **Comparison** - Checks each dependency against all configured bad package lists
5. **Notification** - Shows desktop alert and logs details when matches are found

## Logs

Logs are written to stdout/stderr. When running as a service, redirect to a log file:

```bash
tail -f ~/.dewormer/dewormer.log
```

Log format:

```
2024/11/28 10:30:00 Dewormer started. Scanning every 12h0m0s
2024/11/28 10:30:00 Starting scan...
2024/11/28 10:30:02 Loaded 142 bad packages from 2 lists
2024/11/28 10:30:15 Scan completed in 15.2s. Files scanned: 87
2024/11/28 10:30:15 ⚠️  WARNING: Found 2 infected dependencies!
2024/11/28 10:30:15   - voip-callkit@1.0.2 in /Users/you/projects/app1/package-lock.json (matched: npm-malicious.txt)
```

## Staying Updated

To stay protected against new supply chain attacks:

1. **Subscribe to security feeds** - Follow security researchers and npm/Maven security advisories
2. **Update your lists regularly** - Add newly discovered malicious packages
3. **Share your lists** - Consider contributing to community-maintained bad package lists
4. **Check logs periodically** - Review `~/.dewormer/dewormer.log` to ensure scans are running

## Limitations

- Only detects **known** malicious packages (requires up-to-date bad package lists)
- Does not perform behavioral analysis or detect zero-day attacks
- Requires exact version matches (does not check version ranges)
- Does not scan transitive dependencies from `node_modules` or Maven cache

## Security Considerations

Dewormer is a detection tool, not prevention. When threats are found:

1. **Isolate immediately** - Stop using the affected project
2. **Review the dependency** - Verify it's actually malicious (check CVE databases, security advisories)
3. **Check for compromise** - Scan for signs that malicious code executed
4. **Update and rescan** - Remove the bad dependency and scan again
5. **Report** - Consider reporting to npm/Maven security teams

## Contributing

Contributions welcome! Please submit pull requests for:

- Bug fixes
- New features
- Documentation improvements
- Community-maintained bad package lists

## License

MIT License - See LICENSE file for details

## Acknowledgments

- Uses [beeep](https://github.com/gen2brain/beeep) for cross-platform notifications
- Inspired by the need for better supply chain security awareness

## Support

For issues, questions, or feature requests, please open an issue on GitHub.

---

**Stay safe. Stay updated. Deworm your dependencies.**
