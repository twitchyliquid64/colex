package util

import (
	"io/ioutil"
	"os"
	"path"
	"strconv"
)

// FindProcessesInNamespace returns a list of PIDs for processes in the same
// namespace as the referenced PID.
func FindProcessesInNamespace(pid int) ([]int, error) {
	var out []int

	// get the namespace of the given PID
	ns, err := os.Readlink(path.Join("/proc", strconv.Itoa(pid), "ns", "pid"))
	if err != nil {
		return nil, err
	}

	list, err := ioutil.ReadDir("/proc")
	if err != nil {
		return nil, err
	}
	for _, f := range list {
		if f.IsDir() {
			if pid, _ := strconv.Atoi(f.Name()); pid != 0 {
				candidateNs, err := os.Readlink(path.Join("/proc", strconv.Itoa(pid), "ns", "pid"))
				if err != nil {
					if os.IsNotExist(err) {
						continue // process has exited since we listed /proc
					}
					return nil, err
				}
				if ns == candidateNs {
					out = append(out, pid)
				}
			}
		}
	}
	return out, nil
}
