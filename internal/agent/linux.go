//go:build linux

package agent

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/yikkrrtykj/vps-netwatch/internal/config"
	"github.com/yikkrrtykj/vps-netwatch/internal/model"
)

func LocalStatus(ctx context.Context, cfg config.Agent) model.VPSNodeStatus {
	hostname, _ := os.Hostname()
	name := cfg.Name
	if name == "" {
		name = hostname
	}
	status := model.VPSNodeStatus{
		ID:        cfg.ID,
		Name:      name,
		PublicIP:  cfg.PublicIP,
		Labels:    cfg.Labels,
		CPU:       cpuRatio(),
		Memory:    memoryUsage(),
		Disk:      diskUsage("/"),
		Net:       networkUsage(),
		UptimeSec: uptimeSeconds(),
		Services:  []model.ServiceStatus{systemdStatus(ctx, "sing-box")},
		UpdatedAt: time.Now().UTC(),
	}
	return status
}

func cpuRatio() float64 {
	first, ok := readCPU()
	if !ok {
		return 0
	}
	time.Sleep(200 * time.Millisecond)
	second, ok := readCPU()
	if !ok {
		return 0
	}
	idle := second.idle - first.idle
	total := second.total - first.total
	if total <= 0 {
		return 0
	}
	return float64(total-idle) / float64(total)
}

type cpuSample struct {
	idle  uint64
	total uint64
}

func readCPU() (cpuSample, bool) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return cpuSample{}, false
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return cpuSample{}, false
	}
	fields := strings.Fields(scanner.Text())
	if len(fields) < 5 || fields[0] != "cpu" {
		return cpuSample{}, false
	}
	var sample cpuSample
	for i, field := range fields[1:] {
		value, _ := strconv.ParseUint(field, 10, 64)
		sample.total += value
		if i == 3 || i == 4 {
			sample.idle += value
		}
	}
	return sample, true
}

func memoryUsage() model.ResourceUsage {
	values := readMeminfo()
	total := values["MemTotal"] * 1024
	available := values["MemAvailable"] * 1024
	used := total - available
	return resource(used, total)
}

func readMeminfo() map[string]uint64 {
	out := map[string]uint64{}
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return out
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		value, _ := strconv.ParseUint(fields[1], 10, 64)
		out[key] = value
	}
	return out
}

func diskUsage(path string) model.ResourceUsage {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return model.ResourceUsage{}
	}
	total := stat.Blocks * uint64(stat.Bsize)
	available := stat.Bavail * uint64(stat.Bsize)
	return resource(total-available, total)
}

func networkUsage() model.NetworkUsage {
	file, err := os.Open("/proc/net/dev")
	if err != nil {
		return model.NetworkUsage{}
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	var usage model.NetworkUsage
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		iface := strings.TrimSpace(parts[0])
		if iface == "lo" {
			continue
		}
		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}
		rx, _ := strconv.ParseUint(fields[0], 10, 64)
		tx, _ := strconv.ParseUint(fields[8], 10, 64)
		usage.RXBytes += rx
		usage.TXBytes += tx
	}
	return usage
}

func uptimeSeconds() int64 {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0
	}
	value, _ := strconv.ParseFloat(fields[0], 64)
	return int64(value)
}

func systemdStatus(ctx context.Context, name string) model.ServiceStatus {
	cmd := exec.CommandContext(ctx, "systemctl", "is-active", name)
	output, err := cmd.Output()
	active := strings.TrimSpace(string(output)) == "active"
	status := model.ServiceStatus{Name: name, Active: active}
	if err != nil && len(output) == 0 {
		status.Error = err.Error()
	}
	return status
}

func resource(used, total uint64) model.ResourceUsage {
	var ratio float64
	if total > 0 {
		ratio = float64(used) / float64(total)
	}
	return model.ResourceUsage{Used: used, Total: total, Ratio: ratio}
}

