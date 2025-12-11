# Subnet Scanner

A lightweight Go web application for scanning and pinging IP addresses within a subnet.

## Features

- 🌐 Scan entire subnets using CIDR notation
- 🔄 Ping each IP address 3 times for reliability
- 📊 Visual grid display (up to 25 columns)
- ✅ Real-time online/offline status
- 💾 **Persistent results** - Last scan survives page refreshes and restarts
- 🎨 Modern, responsive UI
- 🐳 Docker support
- ⚙️ Configurable default subnet via environment variable

## Project Structure

```
subnet-scanner/
├── main.go
├── templates/
│   └── index.html
├── Dockerfile
├── docker-compose.yml
├── go.mod (optional - auto-generated if not present)
└── README.md
```

## Setup & Installation

### Option 1: Using Docker Compose (Recommended)

1. Create the project directory structure:
```bash
mkdir -p subnet-scanner/templates
cd subnet-scanner
```

2. Copy all the files to their respective locations:
   - `main.go` → root directory
   - `index.html` → templates/ directory
   - `Dockerfile` → root directory
   - `docker-compose.yml` → root directory

3. Build and run:
```bash
docker-compose up -d
```

4. Access the application at `http://localhost:8080`

### Option 2: Using Docker directly

```bash
# Build the image
docker build -t subnet-scanner .

# Run the container
docker run -d -p 8080:8080 \
  -e DEFAULT_SUBNET=192.168.1.0/24 \
  --name subnet-scanner \
  subnet-scanner
```

### Option 3: Run locally (without Docker)

```bash
# Install Go 1.21+ if not installed
# https://golang.org/dl/

# Initialize Go module (if go.mod doesn't exist)
go mod init subnet-scanner

# Run the application
go run main.go
```

Access at `http://localhost:8080`

## Configuration

### Environment Variables

- `PORT` - Server port (default: `8080`)
- `DEFAULT_SUBNET` - Default subnet to pre-fill (default: `192.168.1.0/24`)
- `ICON` - Path to custom favicon file (optional, e.g., `/app/favicon.ico`)

### Using a Custom Icon

You can set a custom favicon for the web app by providing an icon file:

**Option 1: Using Docker Compose**

1. Place your icon file (e.g., `favicon.ico`, `favicon.png`) in your project directory
2. Update `docker-compose.yml`:

```yaml
services:
  subnet-scanner:
    build: .
    ports:
      - "8080:8080"
    environment:
      - ICON=/app/favicon.ico
    volumes:
      - ./favicon.ico:/app/favicon.ico:ro
```

**Option 2: Using Docker CLI**

```bash
docker run -d -p 8080:8080 \
  -e ICON=/app/favicon.ico \
  -v $(pwd)/favicon.ico:/app/favicon.ico:ro \
  subnet-scanner
```

**Supported Icon Formats:**
- `.ico` (recommended for best browser support)
- `.png` (works in modern browsers)
- `.svg` (works in modern browsers)

If no icon is specified, a default 🌐 emoji icon will be used.

### Example with custom settings:

```bash
docker run -d -p 3000:3000 \
  -e PORT=3000 \
  -e DEFAULT_SUBNET=10.0.0.0/24 \
  subnet-scanner
```

Or in `docker-compose.yml`:
```yaml
environment:
  - PORT=3000
  - DEFAULT_SUBNET=10.0.0.0/24
```

## Network Modes

### Bridge Mode (Default)
The default configuration uses bridge networking. This works for scanning IPs within Docker networks or external networks accessible from the container.

### Host Mode (For Local Network Scanning)
If you need to scan your local network (e.g., your home/office LAN), use host network mode:

In `docker-compose.yml`:
```yaml
services:
  subnet-scanner:
    build: .
    network_mode: host
    environment:
      - DEFAULT_SUBNET=192.168.1.0/24
    # Remove the ports section when using host mode
```

Or with Docker CLI:
```bash
docker run -d --network host \
  -e DEFAULT_SUBNET=192.168.1.0/24 \
  subnet-scanner
```

## Usage

1. Open your browser to `http://localhost:8080`
2. Enter a subnet in CIDR notation (e.g., `192.168.1.0/24`)
3. Click "Start Scan"
4. Wait for the scan to complete (each IP is pinged 3 times)
5. View the results in a visual grid:
   - ✅ Green = Online/Reachable
   - ❌ Red = Offline/Unreachable
6. **Results persist automatically** - refresh the page and your last scan is still there!
7. Click "Scan Again" to perform a new scan
8. Click "Clear Results" to remove saved scan data

### Persistent Storage

The application automatically saves your last scan result using browser storage. This means:
- Results survive page refreshes
- Results persist after Docker container restarts
- Only the most recent scan is kept
- Use "Clear Results" button to remove saved data

## How It Works

1. **Input Validation**: Ensures valid CIDR notation
2. **IP Range Calculation**: Generates list of IPs from subnet (excluding network/broadcast)
3. **Concurrent Pinging**: Pings up to 50 IPs simultaneously for speed
4. **3x Ping Verification**: Each IP is pinged 3 times with 1-second timeout
5. **Results Display**: Shows status in a grid (25 columns max per row)

## Technical Details

- **Backend**: Go 1.21+ (lightweight, ~10MB image)
- **Frontend**: Vanilla JavaScript + CSS (no frameworks)
- **Concurrency**: Goroutines with semaphore pattern for controlled concurrency
- **Ping Command**: Uses OS-native `ping` command (cross-platform)

## Troubleshooting

### Pings not working in Docker?
- Use `network_mode: host` in docker-compose.yml for local network access
- Ensure the container has network access to the target subnet

### Permission issues?
- Docker may need `--cap-add=NET_RAW` for ICMP packets (usually not required with standard ping)

### Large subnets timing out?
- /16 or larger subnets may take several minutes
- Consider breaking into smaller /24 subnets

## Performance

- /24 subnet (254 IPs): ~10-30 seconds
- /16 subnet (65,534 IPs): Several minutes
- Concurrent ping limit: 50 simultaneous connections

## License

MIT License - Feel free to use and modify!

## Contributing

Pull requests welcome! Some ideas for enhancement:
- Port scanning in addition to ping
- Save scan history
- Export results to CSV
- Custom ping count/timeout
- Dark/light theme toggle