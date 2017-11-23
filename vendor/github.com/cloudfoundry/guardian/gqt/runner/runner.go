package runner

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"syscall"
	"time"

	"code.cloudfoundry.org/garden/client"
	"code.cloudfoundry.org/garden/client/connection"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/ginkgomon"
)

type GdnRunnerConfig struct {
	TmpDir string
	User   UserCredential

	Socket2meBin        string
	Socket2meSocketPath string
	RuncRoot            string

	// Garden config
	GdnBin                   string
	TarBin                   string `flag:"tar-bin"`
	InitBin                  string `flag:"init-bin"`
	RuntimePluginBin         string `flag:"runtime-plugin"`
	ImagePluginBin           string `flag:"image-plugin"`
	PrivilegedImagePluginBin string `flag:"privileged-image-plugin"`
	NetworkPluginBin         string `flag:"network-plugin"`
	ExecRunnerBin            string `flag:"dadoo-bin"`
	NSTarBin                 string `flag:"nstar-bin"`

	DefaultRootFS                  string   `flag:"default-rootfs"`
	DepotDir                       string   `flag:"depot"`
	GraphDir                       string   `flag:"graph"`
	ConsoleSocketsPath             string   `flag:"console-sockets-path"`
	BindIP                         string   `flag:"bind-ip"`
	BindPort                       *int     `flag:"bind-port"`
	BindSocket                     string   `flag:"bind-socket"`
	DenyNetworks                   []string `flag:"deny-network"`
	DefaultBlkioWeight             *uint64  `flag:"default-container-blockio-weight"`
	NetworkPluginExtraArgs         []string `flag:"network-plugin-extra-arg"`
	ImagePluginExtraArgs           []string `flag:"image-plugin-extra-arg"`
	RuntimePluginExtraArgs         []string `flag:"runtime-plugin-extra-arg"`
	PrivilegedImagePluginExtraArgs []string `flag:"privileged-image-plugin-extra-arg"`
	MaxContainers                  *uint64  `flag:"max-containers"`
	DebugIP                        string   `flag:"debug-bind-ip"`
	DebugPort                      *int     `flag:"debug-bind-port"`
	PropertiesPath                 string   `flag:"properties-path"`
	PersistentImages               []string `flag:"persistent-image"`
	GraphCleanupThresholdMB        *int     `flag:"graph-cleanup-threshold-in-megabytes"`
	LogLevel                       string   `flag:"log-level"`
	TCPMemoryLimit                 *uint64  `flag:"tcp-memory-limit"`
	CPUQuotaPerShare               *uint64  `flag:"cpu-quota-per-share"`
	IPTablesBin                    string   `flag:"iptables-bin"`
	IPTablesRestoreBin             string   `flag:"iptables-restore-bin"`
	DNSServers                     []string `flag:"dns-server"`
	AdditionalDNSServers           []string `flag:"additional-dns-server"`
	AdditionalHostEntries          []string `flag:"additional-host-entry"`
	MTU                            *int     `flag:"mtu"`
	PortPoolSize                   *int     `flag:"port-pool-size"`
	PortPoolStart                  *int     `flag:"port-pool-start"`
	PortPoolPropertiesPath         string   `flag:"port-pool-properties-path"`
	DestroyContainersOnStartup     *bool    `flag:"destroy-containers-on-startup"`
	DockerRegistry                 string   `flag:"docker-registry"`
	InsecureDockerRegistry         string   `flag:"insecure-docker-registry"`
	AllowHostAccess                *bool    `flag:"allow-host-access"`
	SkipSetup                      *bool    `flag:"skip-setup"`
	UIDMapStart                    *uint32  `flag:"uid-map-start"`
	UIDMapLength                   *uint32  `flag:"uid-map-length"`
	GIDMapStart                    *uint32  `flag:"gid-map-start"`
	GIDMapLength                   *uint32  `flag:"gid-map-length"`
	CleanupProcessDirsOnWait       *bool    `flag:"cleanup-process-dirs-on-wait"`
	DisablePrivilegedContainers    *bool    `flag:"disable-privileged-containers"`
	AppArmor                       string   `flag:"apparmor"`
	Tag                            string   `flag:"tag"`
	NetworkPool                    string   `flag:"network-pool"`

	StartupExpectedToFail bool
}

func (c GdnRunnerConfig) connectionInfo() (string, string) {
	if c.Socket2meSocketPath != "" {
		return "unix", c.Socket2meSocketPath
	}
	if c.BindSocket != "" {
		return "unix", c.BindSocket
	}
	return "tcp", fmt.Sprintf("%s:%d", c.BindIP, *c.BindPort)
}

func (c GdnRunnerConfig) toFlags() []string {
	gardenArgs := []string{}

	vConf := reflect.ValueOf(c)
	tConf := vConf.Type()
	for i := 0; i < tConf.NumField(); i++ {
		tField := tConf.Field(i)
		flagName, ok := tField.Tag.Lookup("flag")
		if !ok {
			continue
		}

		vField := vConf.Field(i)
		if vField.Kind() != reflect.String && vField.IsNil() {
			continue
		}

		fieldVal := reflect.Indirect(vField).Interface()
		switch v := fieldVal.(type) {
		case string:
			if v != "" {
				gardenArgs = append(gardenArgs, "--"+flagName, v)
			}
		case int, uint64, uint32:
			gardenArgs = append(gardenArgs, "--"+flagName, fmt.Sprintf("%d", v))
		case bool:
			if v {
				gardenArgs = append(gardenArgs, "--"+flagName)
			}
		case []string:
			for _, val := range v {
				gardenArgs = append(gardenArgs, "--"+flagName, val)
			}
		default:
			Fail(fmt.Sprintf("unrecognised field type for field %s", flagName))
		}
	}

	return gardenArgs
}

type Binaries struct {
	Tar                   string `json:"tar,omitempty"`
	Gdn                   string `json:"gdn,omitempty"`
	Init                  string `json:"init,omitempty"`
	RuntimePlugin         string `json:"runtime_plugin,omitempty"`
	ImagePlugin           string `json:"image_plugin,omitempty"`
	PrivilegedImagePlugin string `json:"privileged_image_plugin,omitempty"`
	NetworkPlugin         string `json:"network_plugin,omitempty"`
	NoopPlugin            string `json:"noop_plugin,omitempty"`
	ExecRunner            string `json:"execrunner,omitempty"`
	NSTar                 string `json:"nstar,omitempty"`
	Socket2me             string `json:"socket2me,omitempty"`
}

type GardenRunner struct {
	*GdnRunnerConfig
	*ginkgomon.Runner
}

func (r *GardenRunner) Setup() {
	Expect(os.MkdirAll(r.TmpDir, 0755)).To(Succeed())
	Expect(os.MkdirAll(r.DepotDir, 0755)).To(Succeed())
	r.setupDirsForUser()
}

type RunningGarden struct {
	*GardenRunner
	client.Client

	process ifrit.Process
	Pid     int
	logger  lager.Logger
}

const MNT_DETACH = 0x2

var graphDirBase string

func init() {
	graphDirBase = os.Getenv("GARDEN_TEST_GRAPHPATH")
	if graphDirBase == "" {
		// This must be set outside of the Ginkgo node directory (tmpDir) because
		// otherwise the Concourse worker may run into one of the AUFS kernel
		// module bugs that cause the VM to become unresponsive.
		graphDirBase = "/tmp/aufs_mount"
	}
}

func DefaultGdnRunnerConfig() GdnRunnerConfig {
	var config GdnRunnerConfig
	config.Tag = fmt.Sprintf("%d", GinkgoParallelNode())

	var err error
	config.TmpDir, err = ioutil.TempDir("", fmt.Sprintf("test-garden-%s-", config.Tag))
	Expect(os.Chmod(config.TmpDir, 0777)).To(Succeed())
	Expect(err).NotTo(HaveOccurred())

	config.GraphDir = filepath.Join(graphDirBase, fmt.Sprintf("node-%s", config.Tag))
	config.DepotDir = filepath.Join(config.TmpDir, "containers")
	config.ConsoleSocketsPath = filepath.Join(config.TmpDir, "console-sockets")

	if runtime.GOOS == "windows" {
		config.BindIP = "127.0.0.1"
		config.BindPort = intptr(10000 + os.Getpid())
	} else {
		config.BindSocket = fmt.Sprintf("/tmp/garden_%s.sock", config.Tag)
	}

	config.NetworkPool = fmt.Sprintf("10.254.%d.0/22", 4*GinkgoParallelNode())
	config.PortPoolStart = intptr(GinkgoParallelNode() * 7000)

	config.UIDMapStart = uint32ptr(1)
	config.UIDMapLength = uint32ptr(100000)
	config.GIDMapStart = uint32ptr(1)
	config.GIDMapLength = uint32ptr(100000)

	return config
}

func NewGardenRunner(config GdnRunnerConfig) *GardenRunner {
	runner := &GardenRunner{
		GdnRunnerConfig: &config,
		Runner: ginkgomon.New(ginkgomon.Config{
			Name:              "guardian",
			AnsiColorCode:     "31m",
			StartCheck:        "guardian.started",
			StartCheckTimeout: 30 * time.Second,
		}),
	}

	if config.Socket2meSocketPath == "" {
		runner.Command = exec.Command(config.GdnBin, append([]string{"server"}, config.toFlags()...)...)
	} else {
		runner.Command = socket2meCommand(config)
	}

	runner.Command.Env = append(
		os.Environ(),
		[]string{
			fmt.Sprintf("TMPDIR=%s", runner.TmpDir),
			fmt.Sprintf("TEMP=%s", runner.TmpDir),
			fmt.Sprintf("TMP=%s", runner.TmpDir),
		}...,
	)
	if config.RuncRoot != "" {
		runner.Command.Env = append(
			os.Environ(),
			[]string{
				fmt.Sprintf("XDG_RUNTIME_DIR=%s", config.RuncRoot),
			}...,
		)
	}
	setUserCredential(runner)

	runner.Setup()

	return runner
}

func Start(config GdnRunnerConfig) *RunningGarden {
	runner := NewGardenRunner(config)

	gdn := &RunningGarden{
		GardenRunner: runner,
		logger:       lagertest.NewTestLogger("garden-runner"),
	}

	gdn.process = ifrit.Invoke(runner)
	gdn.Pid = runner.Command.Process.Pid
	gdn.Client = client.New(connection.New(runner.connectionInfo()))

	if !config.StartupExpectedToFail {
		Eventually(gdn.Ping, time.Second*10).Should(Succeed())
	}

	return gdn
}

func (r *RunningGarden) Kill() error {
	r.process.Signal(syscall.SIGKILL)
	select {
	case err := <-r.process.Wait():
		return err
	case <-time.After(time.Second * 10):
		r.process.Signal(syscall.SIGKILL)
		return errors.New("timed out waiting for garden to shutdown after 10 seconds")
	}
}

type ErrGardenStop struct {
	error
}

func (r *RunningGarden) DestroyAndStop() error {
	defer r.forceStop()

	if err := r.DestroyContainers(); err != nil {
		return err
	}

	return nil
}

func (r *RunningGarden) forceStop() error {
	if runtime.GOOS == "windows" {
		// Windows doesn't support SIGTERM
		r.Kill()
	} else {
		if err := r.Stop(); err != nil {
			return ErrGardenStop{error: err}
		}
	}

	if err := r.removeTempDirPreservingCgroupMounts(); err != nil {
		return err
	}

	r.Cleanup()

	return nil
}

func (r *RunningGarden) removeTempDirPreservingCgroupMounts() error {
	tmpDir, err := os.Open(r.TmpDir)
	if err != nil {
		return err
	}
	defer tmpDir.Close()
	tmpDirContents, err := tmpDir.Readdir(0)
	if err != nil {
		return err
	}

	for _, tmpDirChild := range tmpDirContents {
		if !strings.Contains(tmpDirChild.Name(), "cgroups-") {
			if err := os.RemoveAll(filepath.Join(r.TmpDir, tmpDirChild.Name())); err != nil {
				return err
			}
		} else {
			cGroupsPath := path.Join(r.TmpDir, tmpDirChild.Name())
			if err = r.cleanGardenCgroups(cGroupsPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *RunningGarden) cleanGardenCgroups(cGroupsPath string) error {
	subsystems, err := ioutil.ReadDir(cGroupsPath)
	if err != nil {
		return err
	}

	for _, subsystem := range subsystems {
		cGroupPath, err := r.findCgroupPath(subsystem.Name())
		if err != nil {
			return err
		}
		nestedCgroup := filepath.Join(cGroupsPath, subsystem.Name(), cGroupPath)
		if err := os.Remove(filepath.Join(nestedCgroup, fmt.Sprintf("garden-%s", r.Tag))); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

func (r *RunningGarden) findCgroupPath(cgroupToFind string) (string, error) {
	cgroupContent, err := ioutil.ReadFile(fmt.Sprintf("/proc/self/cgroup"))
	if err != nil {
		return "", err
	}

	cgroups := strings.Split(string(cgroupContent), "\n")
	for _, cgroup := range cgroups {
		separator := fmt.Sprintf(":%s:", cgroupToFind)
		if strings.Contains(cgroup, separator) {
			return strings.Split(cgroup, separator)[1], nil
		}
	}
	return "", errors.New("Cgroup subsystem not found")
}

func (r *RunningGarden) Stop() error {
	r.process.Signal(syscall.SIGTERM)

	var err error
	for i := 0; i < 5; i++ {
		select {
		case err := <-r.process.Wait():
			return err
		case <-time.After(time.Second * 5):
			r.process.Signal(syscall.SIGTERM)
			err = errors.New("timed out waiting for garden to shutdown after 5 seconds")
		}
	}

	r.process.Signal(syscall.SIGKILL)
	return err
}

func (r *RunningGarden) DestroyContainers() error {
	containers, err := r.Containers(nil)
	if err != nil {
		return err
	}

	for _, container := range containers {
		if err := r.Destroy(container.Handle()); err != nil {
			return err
		}
	}

	return nil
}

type debugVars struct {
	NumGoRoutines int `json:"numGoRoutines"`
}

func (r *RunningGarden) DumpGoroutines() (string, error) {
	debugURL := fmt.Sprintf("http://%s:%d/debug/pprof/goroutine?debug=2", r.DebugIP, *r.DebugPort)
	res, err := http.Get(debugURL)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	return string(b), err
}

func (r *RunningGarden) NumGoroutines() (int, error) {
	debugURL := fmt.Sprintf("http://%s:%d/debug/vars", r.DebugIP, *r.DebugPort)
	res, err := http.Get(debugURL)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	decoder := json.NewDecoder(res.Body)
	var debugVarsData debugVars
	err = decoder.Decode(&debugVarsData)
	if err != nil {
		return 0, err
	}

	return debugVarsData.NumGoRoutines, nil
}

func intptr(i int) *int {
	return &i
}

func uint32ptr(i uint32) *uint32 {
	return &i
}
