package hardware

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"github.com/shirou/gopsutil/v3/process"
)

func GetHardwareInfo() string {
	var info strings.Builder

	hostInfo, err := host.Info()
	if err != nil {
		info.WriteString("Error getting host info\n")
	} else {
		info.WriteString("Host Information:\n")
		info.WriteString(fmt.Sprintf("• Hostname: %s\n", hostInfo.Hostname))
		info.WriteString(fmt.Sprintf("• Platform: %s\n", hostInfo.Platform))
		info.WriteString(fmt.Sprintf("• Platform Family: %s\n", hostInfo.PlatformFamily))
		info.WriteString(fmt.Sprintf("• Boot Time: %s\n", time.Unix(int64(hostInfo.BootTime), 0).Format("2006-01-02 15:04:05")))
		info.WriteString("\n")
	}

	cpuInfo, err := cpu.Info()
	if err != nil {
		info.WriteString("Error getting CPU info\n")
	} else if len(cpuInfo) > 0 {
		cpu := cpuInfo[0]
		info.WriteString("CPU Information:\n")
		info.WriteString(fmt.Sprintf("• Model: %s\n", cpu.ModelName))
		info.WriteString(fmt.Sprintf("• Vendor: %s\n", cpu.VendorID))
		info.WriteString(fmt.Sprintf("• Cores: %d\n", cpu.Cores))
		info.WriteString(fmt.Sprintf("• Cache Size: %d KB\n", cpu.CacheSize))

		if cpu.Mhz > 0 {
			info.WriteString(fmt.Sprintf("• Frequency: %.2f MHz\n", cpu.Mhz))
		}

		info.WriteString("\n")
	}

	memInfo, err := mem.VirtualMemory()
	if err != nil {
		info.WriteString("Error getting memory info\n")
	} else {
		info.WriteString("Memory Information:\n")
		info.WriteString(fmt.Sprintf("• Total: %.2f GB\n", float64(memInfo.Total)/1024/1024/1024))
		info.WriteString(fmt.Sprintf("• Available: %.2f GB\n", float64(memInfo.Available)/1024/1024/1024))
		info.WriteString(fmt.Sprintf("• Used: %.2f GB (%.1f%%)\n", float64(memInfo.Used)/1024/1024/1024, memInfo.UsedPercent))
		info.WriteString("\n")
	}

	info.WriteString("Runtime Information:\n")
	info.WriteString(fmt.Sprintf("• Go Version: %s\n", runtime.Version()))
	info.WriteString(fmt.Sprintf("• OS: %s\n", runtime.GOOS))
	info.WriteString(fmt.Sprintf("• Architecture: %s\n", runtime.GOARCH))
	info.WriteString(fmt.Sprintf("• NumCPU: %d\n", runtime.NumCPU()))
	info.WriteString(fmt.Sprintf("• NumGoroutine: %d\n", runtime.NumGoroutine()))

	info.WriteString(getGPUInfo())

	info.WriteString(getDiskInfo())

	info.WriteString(getNetworkInfo())

	info.WriteString(getProcessInfo())

	return info.String()
}

func getGPUInfo() string {
	var info strings.Builder

	info.WriteString("GPU Information:\n")

	gpuInfo := getGPUInfoWindows()
	if gpuInfo != "" {
		info.WriteString(gpuInfo)
	} else {
		info.WriteString("• GPU detection not available on this platform\n")
	}
	info.WriteString("\n")

	return info.String()
}

func getGPUInfoWindows() string {
	var info strings.Builder

	cmd := exec.Command("powershell", "-Command", "Get-WmiObject -Class Win32_VideoController | Select-Object Name, DriverVersion, AdapterRAM | Format-List")
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		lines := strings.Split(string(output), "\n")
		gpuCount := 0

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Name") && strings.Contains(line, ":") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					gpuName := strings.TrimSpace(parts[1])
					if gpuName != "" && gpuName != "Name" {
						gpuCount++
						info.WriteString(fmt.Sprintf("• GPU %d: %s\n", gpuCount, gpuName))
					}
				}
			}
		}

		if gpuCount > 0 {
			return info.String()
		}
	}

	cmd = exec.Command("wmic", "path", "win32_VideoController", "get", "name", "/format:list")
	output, err = cmd.Output()
	if err == nil && len(output) > 0 {
		lines := strings.Split(string(output), "\n")
		gpuCount := 0

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "Name=") && line != "Name=" {
				gpuName := strings.TrimPrefix(line, "Name=")
				if gpuName != "" {
					gpuCount++
					info.WriteString(fmt.Sprintf("• GPU %d: %s\n", gpuCount, gpuName))
				}
			}
		}

		if gpuCount > 0 {
			return info.String()
		}
	}

	cmd = exec.Command("powershell", "-Command", "Get-WmiObject Win32_VideoController | Select-Object Name")
	output, err = cmd.Output()
	if err == nil && len(output) > 0 {
		lines := strings.Split(string(output), "\n")
		gpuCount := 0

		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line != "" && line != "Name" && !strings.Contains(line, "----") {
				gpuCount++
				info.WriteString(fmt.Sprintf("• GPU %d: %s\n", gpuCount, line))
			}
		}

		if gpuCount > 0 {
			return info.String()
		}
	}

	info.WriteString("• GPU detection failed - no accessible method found\n")
	return info.String()
}

func getDiskInfo() string {
	var info strings.Builder

	info.WriteString("Disk Information:\n")

	partitions, err := disk.Partitions(false)
	if err != nil {
		info.WriteString("Error getting disk partitions\n")
		return info.String()
	}

	for i, partition := range partitions {
		if i >= 5 {
			info.WriteString("• ... (additional partitions hidden)\n")
			break
		}

		usage, err := disk.Usage(partition.Mountpoint)
		if err != nil {
			continue
		}

		info.WriteString(fmt.Sprintf("• %s (%s): %.2f GB / %.2f GB (%.1f%% used)\n",
			partition.Device,
			partition.Fstype,
			float64(usage.Used)/1024/1024/1024,
			float64(usage.Total)/1024/1024/1024,
			usage.UsedPercent,
		))
	}
	info.WriteString("\n")

	return info.String()
}

func getNetworkInfo() string {
	var info strings.Builder

	info.WriteString("Network Information:\n")

	interfaces, err := net.Interfaces()
	if err != nil {
		info.WriteString("Error getting network interfaces\n")
		return info.String()
	}

	activeInterfaces := 0
	for _, iface := range interfaces {
		if len(iface.Addrs) > 0 {
			activeInterfaces++
			if activeInterfaces <= 3 {
				info.WriteString(fmt.Sprintf("• %s: %s\n", iface.Name, iface.HardwareAddr))
				for _, addr := range iface.Addrs {
					info.WriteString(fmt.Sprintf("  - %s\n", addr.Addr))
				}
			}
		}
	}

	if activeInterfaces > 3 {
		info.WriteString(fmt.Sprintf("• ... (%d more interfaces)\n", activeInterfaces-3))
	}
	info.WriteString("\n")

	return info.String()
}

func getProcessInfo() string {
	var info strings.Builder

	info.WriteString("System Load:\n")

	processes, err := process.Processes()
	if err != nil {
		info.WriteString("Error getting process count\n")
		return info.String()
	}

	info.WriteString(fmt.Sprintf("• Running Processes: %d\n", len(processes)))

	cpuPercent, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		info.WriteString(fmt.Sprintf("• CPU Usage: %.1f%%\n", cpuPercent[0]))
	}

	info.WriteString("\n")

	return info.String()
}
