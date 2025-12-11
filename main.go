package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"sync"
)

//go:embed templates/*
var templateFS embed.FS

type ScanResult struct {
	IP    string `json:"ip"`
	Alive bool   `json:"alive"`
}

type ScanRequest struct {
	Subnet string `json:"subnet"`
}

var defaultSubnet string
var iconPath string

func init() {
	defaultSubnet = os.Getenv("DEFAULT_SUBNET")
	if defaultSubnet == "" {
		defaultSubnet = "192.168.1.0/24"
	}
	
	iconPath = os.Getenv("ICON")
}

func main() {
	http.HandleFunc("/", indexHandler)
	http.HandleFunc("/scan", scanHandler)
	http.HandleFunc("/favicon.ico", faviconHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s with default subnet: %s\n", port, defaultSubnet)
	if iconPath != "" {
		log.Printf("Using custom icon: %s\n", iconPath)
	}
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFS(templateFS, "templates/index.html")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := map[string]string{
		"DefaultSubnet": defaultSubnet,
		"HasIcon":       fmt.Sprintf("%t", iconPath != ""),
	}

	tmpl.Execute(w, data)
}

func scanHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ScanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ips, err := getIPsFromSubnet(req.Subnet)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	results := scanIPs(ips)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	if iconPath == "" {
		http.NotFound(w, r)
		return
	}

	// Check if file exists
	if _, err := os.Stat(iconPath); os.IsNotExist(err) {
		http.NotFound(w, r)
		return
	}

	// Serve the icon file
	http.ServeFile(w, r, iconPath)
}

func getIPsFromSubnet(subnet string) ([]string, error) {
	ip, ipnet, err := net.ParseCIDR(subnet)
	if err != nil {
		return nil, fmt.Errorf("invalid CIDR notation: %v", err)
	}

	var ips []string
	for ip := ip.Mask(ipnet.Mask); ipnet.Contains(ip); incIP(ip) {
		ips = append(ips, ip.String())
	}

	// Remove network and broadcast addresses
	if len(ips) > 2 {
		ips = ips[1 : len(ips)-1]
	}

	return ips, nil
}

func incIP(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}

func scanIPs(ips []string) []ScanResult {
	results := make([]ScanResult, len(ips))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 50) // Limit concurrent pings

	for i, ip := range ips {
		wg.Add(1)
		go func(idx int, ipAddr string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			alive := pingIP(ipAddr)
			results[idx] = ScanResult{
				IP:    ipAddr,
				Alive: alive,
			}
		}(i, ip)
	}

	wg.Wait()

	// Sort results by IP address
	sort.Slice(results, func(i, j int) bool {
		return ipToInt(results[i].IP) < ipToInt(results[j].IP)
	})

	return results
}

func pingIP(ip string) bool {
	var cmd *exec.Cmd
	
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "-n", "3", "-w", "1000", ip)
	} else {
		cmd = exec.Command("ping", "-c", "3", "-W", "1", ip)
	}

	err := cmd.Run()
	return err == nil
}

func ipToInt(ip string) uint32 {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return 0
	}
	var result uint32
	for i, part := range parts {
		var octet uint32
		fmt.Sscanf(part, "%d", &octet)
		result |= octet << uint(24-i*8)
	}
	return result
}