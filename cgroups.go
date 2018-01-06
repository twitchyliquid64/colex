package colex

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// CGroupConfig represents configuration for the cgroups.
type CGroupConfig struct {
	CPUTimeBetweenPeriodsUS int64
	CPUTimePerPeriodUS      int64
	MemoryMaxBytes          int64
}

// NewCGroupSet creates a suite of Cgroups with the process within it.
func NewCGroupSet(id string, pid int, c *CGroupConfig) (*CGroupSet, error) {
	s := CGroupSet{ID: "c" + id, PID: pid}

	cpuPath, err := newCPUGroup(s.ID, pid)
	if err != nil {
		return nil, err
	}
	s.CPUGroupPath = cpuPath
	if err2 := setCPUGroupStats(cpuPath, c); err2 != nil {
		s.Close()
		return nil, err2
	}

	memPath, err := newMemoryGroup(s.ID, pid, c)
	if err != nil {
		return nil, err
	}
	s.MemoryGroupPath = memPath

	return &s, nil
}

// CGroupSet represents a process within (a) cgroup(s).
type CGroupSet struct {
	ID              string
	PID             int
	CPUGroupPath    string
	MemoryGroupPath string
}

// Close destroys all cgroups in this set.
func (s *CGroupSet) Close() error {
	for _, path := range []string{s.CPUGroupPath, s.MemoryGroupPath} {
		if path != "" {
			os.RemoveAll(path)
			if _, err := os.Stat(path); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}

	return nil
}

func newMemoryGroup(id string, pid int, c *CGroupConfig) (string, error) {
	subsystemPath, err := GetSubsystemMountpoint("memory")
	if err != nil {
		return "", err
	}
	path := filepath.Join(subsystemPath, id)

	if err := os.MkdirAll(path, 0755); err != nil && !os.IsExist(err) {
		return "", err
	}

	if c.MemoryMaxBytes != 0 {
		if err := writeValue(path, "memory.limit_in_bytes", strconv.FormatInt(c.MemoryMaxBytes, 10)); err != nil {
			return "", err
		}
	}

	if err := writeValue(path, "cgroup.procs", strconv.Itoa(pid)); err != nil {
		return "", err
	}
	return path, nil
}

func newCPUGroup(id string, pid int) (string, error) {
	subsystemPath, err := GetSubsystemMountpoint("cpu")
	if err != nil {
		return "", err
	}

	path := filepath.Join(subsystemPath, id)

	if err := os.MkdirAll(path, 0755); err != nil && !os.IsExist(err) {
		return "", err
	}

	if err := writeValue(path, "cgroup.procs", strconv.Itoa(pid)); err != nil {
		return "", err
	}
	return path, nil
}

func setCPUGroupStats(path string, c *CGroupConfig) error {
	if c.CPUTimeBetweenPeriodsUS != 0 {
		if err := writeValue(path, "cpu.cfs_period_us", strconv.FormatInt(c.CPUTimeBetweenPeriodsUS, 10)); err != nil {
			return err
		}
	}
	if c.CPUTimePerPeriodUS != 0 {
		fmt.Println(c.CPUTimePerPeriodUS)
		if err := writeValue(path, "cpu.cfs_quota_us", strconv.FormatInt(c.CPUTimePerPeriodUS, 10)); err != nil {
			return err
		}
	}

	return nil
}

// CREDIT FOR THE BELOW: https://github.com/0xef53/go-cgroup

func writeValue(dir, file, data string) error {
	return ioutil.WriteFile(filepath.Join(dir, file), []byte(data), 0700)
}

func readInt64Value(dir, file string) (int64, error) {
	c, err := ioutil.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return 0, err
	}
	return strconv.ParseInt(strings.TrimSpace(string(c)), 10, 64)
}

func parsePairValue(s string) (string, uint64, error) {
	parts := strings.Fields(s)
	switch len(parts) {
	case 2:
		value, err := strconv.ParseUint(parts[1], 10, 64)
		if err != nil {
			return "", 0, fmt.Errorf("Unable to convert param value (%q) to uint64: %v", parts[1], err)
		}

		return parts[0], value, nil
	default:
		return "", 0, fmt.Errorf("incorrect key-value format: %s", s)
	}
}

// GetEnabledSubsystems returns a map with the all supported by the kernel control group subsystems.
func GetEnabledSubsystems() (map[string]int, error) {
	cgroupsFile, err := os.Open("/proc/cgroups")
	if err != nil {
		return nil, err
	}
	defer cgroupsFile.Close()

	scanner := bufio.NewScanner(cgroupsFile)

	// Skip the first line. It's a comment
	scanner.Scan()

	cgroups := make(map[string]int)
	for scanner.Scan() {
		var subsystem string
		var hierarchy int
		var num int
		var enabled int
		fmt.Sscanf(scanner.Text(), "%s %d %d %d", &subsystem, &hierarchy, &num, &enabled)

		if enabled == 1 {
			cgroups[subsystem] = hierarchy
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("Cannot parsing /proc/cgroups: %s", err)
	}

	return cgroups, nil
}

// GetSubsystemMountpoint returns the path where a given subsystem is mounted.
func GetSubsystemMountpoint(subsystem string) (string, error) {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		txt := scanner.Text()
		fields := strings.Split(txt, " ")
		for _, opt := range strings.Split(fields[len(fields)-1], ",") {
			if opt == subsystem {
				return fields[4], nil
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("Mountpoint not found: %s", subsystem)
}
