// Package pfsense provides system metrics collection for pfSense/FreeBSD.
package pfsense

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Metrics contains system metrics from pfSense.
type Metrics struct {
	Hostname   string
	CPU        float64 // CPU usage percentage
	MemUsed    uint64  // Memory used in bytes
	MemTotal   uint64  // Total memory in bytes
	Uptime     time.Duration
	LoadAvg    [3]float64 // 1, 5, 15 minute load averages
	Interfaces []InterfaceMetrics
}

// InterfaceMetrics contains network interface statistics.
type InterfaceMetrics struct {
	Name        string
	Description string // e.g., "WAN", "INTERNAL_LAN"
	Status      string // active, no carrier
	IP          string
	Netmask     string
	RxBytes     uint64
	TxBytes     uint64
}

// MetricsProvider is an interface for collecting system metrics.
type MetricsProvider interface {
	GetMetrics() (*Metrics, error)
}

// SystemMetrics implements MetricsProvider for FreeBSD/pfSense.
type SystemMetrics struct {
	prevCPU cpuStats
}

type cpuStats struct {
	user   uint64
	nice   uint64
	system uint64
	intr   uint64
	idle   uint64
	total  uint64
}

// NewSystemMetrics creates a new SystemMetrics collector.
func NewSystemMetrics() *SystemMetrics {
	return &SystemMetrics{}
}

// GetMetrics collects current system metrics.
func (s *SystemMetrics) GetMetrics() (*Metrics, error) {
	m := &Metrics{}

	// Get hostname
	hostname, err := os.Hostname()
	if err == nil {
		m.Hostname = hostname
	}

	// Get uptime
	uptime, err := s.getUptime()
	if err == nil {
		m.Uptime = uptime
	}

	// Get CPU usage
	cpu, err := s.getCPU()
	if err == nil {
		m.CPU = cpu
	}

	// Get memory
	memUsed, memTotal, err := s.getMemory()
	if err == nil {
		m.MemUsed = memUsed
		m.MemTotal = memTotal
	}

	// Get load average
	load, err := s.getLoadAvg()
	if err == nil {
		m.LoadAvg = load
	}

	// Get network interfaces
	interfaces, err := s.getInterfaces()
	if err == nil {
		m.Interfaces = interfaces
	}

	return m, nil
}

// getUptime returns the system uptime.
func (s *SystemMetrics) getUptime() (time.Duration, error) {
	// Try sysctl (FreeBSD/pfSense)
	out, err := exec.Command("sysctl", "-n", "kern.boottime").Output()
	if err == nil {
		// Parse: { sec = 1234567890, usec = 123456 }
		str := strings.TrimSpace(string(out))
		var sec int64
		_, err := fmt.Sscanf(str, "{ sec = %d,", &sec)
		if err == nil && sec > 0 {
			bootTime := time.Unix(sec, 0)
			return time.Since(bootTime), nil
		}
	}

	// Try /proc/uptime (Linux)
	data, err := os.ReadFile("/proc/uptime")
	if err == nil {
		parts := strings.Fields(string(data))
		if len(parts) > 0 {
			seconds, err := strconv.ParseFloat(parts[0], 64)
			if err == nil {
				return time.Duration(seconds * float64(time.Second)), nil
			}
		}
	}

	return 0, fmt.Errorf("unable to get uptime")
}

// getCPU returns CPU usage percentage.
func (s *SystemMetrics) getCPU() (float64, error) {
	// Try sysctl (FreeBSD)
	out, err := exec.Command("sysctl", "-n", "kern.cp_time").Output()
	if err == nil {
		parts := strings.Fields(strings.TrimSpace(string(out)))
		if len(parts) >= 5 {
			user, _ := strconv.ParseUint(parts[0], 10, 64)
			nice, _ := strconv.ParseUint(parts[1], 10, 64)
			sys, _ := strconv.ParseUint(parts[2], 10, 64)
			intr, _ := strconv.ParseUint(parts[3], 10, 64)
			idle, _ := strconv.ParseUint(parts[4], 10, 64)

			total := user + nice + sys + intr + idle
			if s.prevCPU.total > 0 {
				deltaTotal := total - s.prevCPU.total
				deltaIdle := idle - s.prevCPU.idle
				if deltaTotal > 0 {
					usage := 100.0 * float64(deltaTotal-deltaIdle) / float64(deltaTotal)
					s.prevCPU = cpuStats{user, nice, sys, intr, idle, total}
					return usage, nil
				}
			}
			s.prevCPU = cpuStats{user, nice, sys, intr, idle, total}
			return 0, nil
		}
	}

	// Try /proc/stat (Linux)
	data, err := os.ReadFile("/proc/stat")
	if err == nil {
		lines := strings.Split(string(data), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "cpu ") {
				parts := strings.Fields(line)
				if len(parts) >= 5 {
					user, _ := strconv.ParseUint(parts[1], 10, 64)
					nice, _ := strconv.ParseUint(parts[2], 10, 64)
					sys, _ := strconv.ParseUint(parts[3], 10, 64)
					idle, _ := strconv.ParseUint(parts[4], 10, 64)
					total := user + nice + sys + idle

					if s.prevCPU.total > 0 {
						deltaTotal := total - s.prevCPU.total
						deltaIdle := idle - s.prevCPU.idle
						if deltaTotal > 0 {
							usage := 100.0 * float64(deltaTotal-deltaIdle) / float64(deltaTotal)
							s.prevCPU = cpuStats{user: user, nice: nice, system: sys, idle: idle, total: total}
							return usage, nil
						}
					}
					s.prevCPU = cpuStats{user: user, nice: nice, system: sys, idle: idle, total: total}
					return 0, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("unable to get CPU stats")
}

// getMemory returns memory usage.
func (s *SystemMetrics) getMemory() (used, total uint64, err error) {
	// Try sysctl (FreeBSD)
	pageSize := uint64(4096)
	psOut, err := exec.Command("sysctl", "-n", "hw.pagesize").Output()
	if err == nil {
		if ps, err := strconv.ParseUint(strings.TrimSpace(string(psOut)), 10, 64); err == nil {
			pageSize = ps
		}
	}

	memOut, err := exec.Command("sysctl", "-n", "hw.physmem").Output()
	if err == nil {
		if mem, err := strconv.ParseUint(strings.TrimSpace(string(memOut)), 10, 64); err == nil {
			total = mem
		}
	}

	freeOut, err := exec.Command("sysctl", "-n", "vm.stats.vm.v_free_count").Output()
	if err == nil {
		if free, err := strconv.ParseUint(strings.TrimSpace(string(freeOut)), 10, 64); err == nil {
			freeBytes := free * pageSize
			if total > freeBytes {
				used = total - freeBytes
			}
			return used, total, nil
		}
	}

	// Try /proc/meminfo (Linux)
	f, err := os.Open("/proc/meminfo")
	if err != nil {
		return 0, 0, err
	}
	defer f.Close()

	var memTotal, memAvailable uint64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "MemTotal:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				kb, _ := strconv.ParseUint(parts[1], 10, 64)
				memTotal = kb * 1024
			}
		} else if strings.HasPrefix(line, "MemAvailable:") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				kb, _ := strconv.ParseUint(parts[1], 10, 64)
				memAvailable = kb * 1024
			}
		}
	}

	if memTotal > 0 {
		return memTotal - memAvailable, memTotal, nil
	}

	return 0, 0, fmt.Errorf("unable to get memory stats")
}

// getLoadAvg returns system load averages.
func (s *SystemMetrics) getLoadAvg() ([3]float64, error) {
	var load [3]float64

	// Try sysctl (FreeBSD)
	out, err := exec.Command("sysctl", "-n", "vm.loadavg").Output()
	if err == nil {
		// Parse: { 0.50 0.75 1.00 }
		str := strings.Trim(strings.TrimSpace(string(out)), "{}")
		parts := strings.Fields(str)
		if len(parts) >= 3 {
			load[0], _ = strconv.ParseFloat(parts[0], 64)
			load[1], _ = strconv.ParseFloat(parts[1], 64)
			load[2], _ = strconv.ParseFloat(parts[2], 64)
			return load, nil
		}
	}

	// Try /proc/loadavg (Linux)
	data, err := os.ReadFile("/proc/loadavg")
	if err == nil {
		parts := strings.Fields(string(data))
		if len(parts) >= 3 {
			load[0], _ = strconv.ParseFloat(parts[0], 64)
			load[1], _ = strconv.ParseFloat(parts[1], 64)
			load[2], _ = strconv.ParseFloat(parts[2], 64)
			return load, nil
		}
	}

	return load, fmt.Errorf("unable to get load average")
}

// getInterfaces returns network interface information by parsing ifconfig output.
func (s *SystemMetrics) getInterfaces() ([]InterfaceMetrics, error) {
	var result []InterfaceMetrics

	// Run ifconfig to get interface details including descriptions
	out, err := exec.Command("ifconfig").Output()
	if err != nil {
		return nil, err
	}

	var current *InterfaceMetrics
	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		// New interface starts with name at beginning of line (not whitespace)
		if len(line) > 0 && line[0] != '\t' && line[0] != ' ' {
			// Save previous interface
			if current != nil && current.Name != "" {
				// Skip loopback, pflog, pfsync, enc
				if !strings.HasPrefix(current.Name, "lo") &&
					!strings.HasPrefix(current.Name, "pflog") &&
					!strings.HasPrefix(current.Name, "pfsync") &&
					!strings.HasPrefix(current.Name, "enc") {
					// Get traffic stats
					rx, tx := s.getInterfaceStats(current.Name)
					current.RxBytes = rx
					current.TxBytes = tx
					result = append(result, *current)
				}
			}

			// Start new interface
			parts := strings.Split(line, ":")
			if len(parts) > 0 {
				current = &InterfaceMetrics{
					Name:   strings.TrimSpace(parts[0]),
					Status: "down",
				}
			}
		} else if current != nil && len(line) > 0 {
			line = strings.TrimSpace(line)

			// Parse description
			if strings.HasPrefix(line, "description:") {
				current.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			}

			// Parse status (active or no carrier)
			if strings.HasPrefix(line, "status:") {
				status := strings.TrimSpace(strings.TrimPrefix(line, "status:"))
				current.Status = status
			}

			// Parse inet address
			if strings.HasPrefix(line, "inet ") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					current.IP = parts[1]
				}
				if len(parts) >= 4 && parts[2] == "netmask" {
					current.Netmask = parts[3]
				}
			}
		}
	}

	// Don't forget the last interface
	if current != nil && current.Name != "" {
		if !strings.HasPrefix(current.Name, "lo") &&
			!strings.HasPrefix(current.Name, "pflog") &&
			!strings.HasPrefix(current.Name, "pfsync") &&
			!strings.HasPrefix(current.Name, "enc") {
			rx, tx := s.getInterfaceStats(current.Name)
			current.RxBytes = rx
			current.TxBytes = tx
			result = append(result, *current)
		}
	}

	return result, nil
}

// getInterfaceStats gets RX/TX bytes for an interface using netstat.
func (s *SystemMetrics) getInterfaceStats(name string) (rx, tx uint64) {
	// Use netstat -ibn to get interface statistics
	// Output format: Name Mtu Network Address Ipkts Ierrs Idrop Ibytes Opkts Oerrs Obytes Coll
	//                0    1   2       3       4     5     6     7      8     9     10     11
	out, err := exec.Command("netstat", "-ibn").Output()
	if err != nil {
		return 0, 0
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) < 11 {
			continue
		}

		// Check if this line matches our interface (exact match or with Link# entry)
		ifaceName := parts[0]
		// Remove trailing * for inactive interfaces
		ifaceName = strings.TrimSuffix(ifaceName, "*")

		if ifaceName != name {
			continue
		}

		// Skip lines with fe80 or non-link entries (they have "-" in packet columns)
		// We want the <Link#N> lines which have the total byte counts
		if !strings.Contains(parts[2], "<Link#") {
			continue
		}

		// Parse Ibytes (column 7) and Obytes (column 10)
		rx, _ = strconv.ParseUint(parts[7], 10, 64)
		tx, _ = strconv.ParseUint(parts[10], 10, 64)
		return rx, tx
	}

	return 0, 0
}

// FormatBytes formats bytes to a human-readable string.
func FormatBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// FormatRate formats bytes per second to a human-readable rate.
func FormatRate(bytesPerSec float64) string {
	if bytesPerSec < 1024 {
		return fmt.Sprintf("%.0f B/s", bytesPerSec)
	}
	if bytesPerSec < 1024*1024 {
		return fmt.Sprintf("%.1f KB/s", bytesPerSec/1024)
	}
	return fmt.Sprintf("%.1f MB/s", bytesPerSec/1024/1024)
}
