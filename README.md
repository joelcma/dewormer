# Dewormer

A cross-platform background scanner for detecting compromised npm and Maven dependencies in your projects.

## What It Does

Dewormer continuously monitors your development directories for known malicious packages from supply chain attacks. It scans `package-lock.json` and `pom.xml` files, compares dependencies against curated lists of compromised packages, and alerts you via desktop notifications when threats are detected.

**Install it. Configure it. Forget it.**

## Features

- 🔍 **Automatic scanning** - Runs in the background at configurable intervals (default: every 12 hours)
- 🔔 **Desktop notifications** - Get alerted immediately when threats are found
- 📦 **Multi-ecosystem support** - Scans both npm (package-lock.json) and Maven (pom.xml)
- 🎯 **Customizable** - Configure scan paths and maintain your own bad package lists
- 🪶 **Lightweight** - Single binary, minimal resource usage
- 🖥️ **Cross-platform** - Works on Windows, macOS, and Linux

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
  ],
  "scan_interval": "12h"
}
```

**Configuration options:**

- `scan_paths` - List of directories to scan recursively for dependency files
- `bad_package_lists` - List of text files containing known malicious packages
- `scan_interval` - How often to scan (e.g., "6h", "12h", "24h", "30m")

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

Create `/etc/systemd/system/dewormer.service`:

```ini
[Unit]
Description=Dewormer Dependency Scanner
After=network.target

[Service]
Type=simple
User=yourname
ExecStart=/usr/local/bin/dewormer
Restart=always

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable dewormer
sudo systemctl start dewormer
sudo systemctl status dewormer
```

#### Windows (Task Scheduler)

1. Open Task Scheduler
2. Create Basic Task → Name it "Dewormer"
3. Trigger: "When I log on"
4. Action: "Start a program" → Browse to `dewormer.exe`
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
