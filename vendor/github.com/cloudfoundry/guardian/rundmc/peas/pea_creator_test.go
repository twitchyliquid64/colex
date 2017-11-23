package peas_test

import (
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/guardian/logging"
	"code.cloudfoundry.org/guardian/rundmc/depot/depotfakes"
	"code.cloudfoundry.org/guardian/rundmc/goci"
	"code.cloudfoundry.org/guardian/rundmc/peas"
	"code.cloudfoundry.org/guardian/rundmc/peas/peasfakes"
	"code.cloudfoundry.org/guardian/rundmc/runrunc"
	"code.cloudfoundry.org/guardian/rundmc/runrunc/runruncfakes"
	"code.cloudfoundry.org/guardian/rundmc/signals/signalsfakes"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	specs "github.com/opencontainers/runtime-spec/specs-go"
)

var _ = Describe("PeaCreator", func() {
	const imageURI = "some-image-uri"

	var (
		volumizer        *peasfakes.FakeVolumizer
		pidGetter        *peasfakes.FakePidGetter
		bundleGenerator  *depotfakes.FakeBundleGenerator
		bundleSaver      *depotfakes.FakeBundleSaver
		processBuilder   *runruncfakes.FakeProcessBuilder
		containerCreator *peasfakes.FakeContainerCreator
		signallerFactory *peasfakes.FakeSignallerFactory
		signaller        *signalsfakes.FakeSignaller

		peaCreator *peas.PeaCreator

		ctrHandle    string
		ctrBundleDir string
		log          *lagertest.TestLogger

		generatedBundle = goci.Bndl{Spec: specs.Spec{Version: "our-bundle"}}
		builtProcess    = &runrunc.PreparedSpec{
			Process: specs.Process{Cwd: "some-cwd"},
		}
		processSpec garden.ProcessSpec
	)

	BeforeEach(func() {
		volumizer = new(peasfakes.FakeVolumizer)
		volumizer.CreateReturns(specs.Spec{Version: "some-spec-version"}, nil)
		pidGetter = new(peasfakes.FakePidGetter)
		pidGetter.PidReturns(123, nil)
		bundleGenerator = new(depotfakes.FakeBundleGenerator)
		bundleGenerator.GenerateReturns(generatedBundle, nil)
		bundleSaver = new(depotfakes.FakeBundleSaver)
		processBuilder = new(runruncfakes.FakeProcessBuilder)
		processBuilder.BuildProcessReturns(builtProcess)
		containerCreator = new(peasfakes.FakeContainerCreator)
		signallerFactory = new(peasfakes.FakeSignallerFactory)
		signaller = new(signalsfakes.FakeSignaller)
		signallerFactory.NewSignallerReturns(signaller)

		peaCreator = &peas.PeaCreator{
			Volumizer:        volumizer,
			PidGetter:        pidGetter,
			BundleGenerator:  bundleGenerator,
			BundleSaver:      bundleSaver,
			ProcessBuilder:   processBuilder,
			ContainerCreator: containerCreator,
			SignallerFactory: signallerFactory,
		}

		var err error
		ctrHandle = "pea-creator-tests"
		ctrBundleDir, err = ioutil.TempDir("", "pea-creator-tests")
		Expect(err).NotTo(HaveOccurred())
		log = lagertest.NewTestLogger("peas-unit-tests")
		processSpec = garden.ProcessSpec{
			ID:   "some-id",
			Dir:  "/some/dir",
			User: "4:5",
			Image: garden.ImageRef{
				URI:      imageURI,
				Username: "cakeuser",
				Password: "cakepassword",
			},
		}
	})

	AfterEach(func() {
		Expect(os.RemoveAll(ctrBundleDir)).To(Succeed())
	})

	Describe("pea creation succeeding", func() {
		var process garden.Process

		JustBeforeEach(func() {
			var err error
			process, err = peaCreator.CreatePea(log, processSpec, garden.ProcessIO{}, ctrHandle, ctrBundleDir)
			Expect(err).NotTo(HaveOccurred())
		})

		It("creates the bundle directory", func() {
			Expect(filepath.Join(ctrBundleDir, "processes", processSpec.ID)).To(BeADirectory())
		})

		It("creates a volume", func() {
			Expect(volumizer.CreateCallCount()).To(Equal(1))
			_, actualSpec := volumizer.CreateArgsForCall(0)
			Expect(actualSpec.Handle).To(Equal(processSpec.ID))
			Expect(actualSpec.Image).To(Equal(garden.ImageRef{
				URI:      imageURI,
				Username: "cakeuser",
				Password: "cakepassword",
			}))
		})

		It("passes the processID as handle to the bundle generator", func() {
			Expect(bundleGenerator.GenerateCallCount()).To(Equal(1))
			actualCtrSpec, _ := bundleGenerator.GenerateArgsForCall(0)
			Expect(actualCtrSpec.Handle).To(Equal(processSpec.ID))
		})

		It("generates a runtime spec from the VolumeCreator's runtimeSpec", func() {
			Expect(bundleGenerator.GenerateCallCount()).To(Equal(1))
			actualCtrSpec, _ := bundleGenerator.GenerateArgsForCall(0)
			Expect(actualCtrSpec.BaseConfig).To(Equal(specs.Spec{Version: "some-spec-version"}))
		})

		It("passes the container handle as cgroup path to the bundle generator", func() {
			Expect(bundleGenerator.GenerateCallCount()).To(Equal(1))
			actualCtrSpec, _ := bundleGenerator.GenerateArgsForCall(0)
			Expect(actualCtrSpec.CgroupPath).To(Equal(ctrHandle))
		})

		It("shares all namespaces apart from mnt with the container", func() {
			Expect(bundleGenerator.GenerateCallCount()).To(Equal(1))
			actualCtrSpec, _ := bundleGenerator.GenerateArgsForCall(0)
			Expect(actualCtrSpec.Namespaces).To(Equal(map[string]string{
				"mount":   "",
				"network": "/proc/123/ns/net",
				"user":    "/proc/123/ns/user",
				"ipc":     "/proc/123/ns/ipc",
				"pid":     "/proc/123/ns/pid",
				"uts":     "/proc/123/ns/uts",
			}))
		})

		It("passes the ctrBundlePath to the bundle generator", func() {
			Expect(bundleGenerator.GenerateCallCount()).To(Equal(1))
			_, actualCtrBundle := bundleGenerator.GenerateArgsForCall(0)
			Expect(actualCtrBundle).To(Equal(ctrBundleDir))
		})

		It("builds a process", func() {
			Expect(processBuilder.BuildProcessCallCount()).To(Equal(1))
			actualBundle, actualProcessSpec := processBuilder.BuildProcessArgsForCall(0)
			Expect(actualBundle).To(Equal(generatedBundle))
			Expect(actualProcessSpec).To(Equal(runrunc.ProcessSpec{
				ProcessSpec:  processSpec,
				ContainerUID: 4,
				ContainerGID: 5,
			}))
		})

		It("saves the bundle (containing the built process) to disk", func() {
			Expect(bundleSaver.SaveCallCount()).To(Equal(1))
			actualBundle, actualBundlePath := bundleSaver.SaveArgsForCall(0)
			expectedBundle := generatedBundle.WithProcess(specs.Process{Cwd: "some-cwd"})
			Expect(actualBundle).To(Equal(expectedBundle))
			Expect(actualBundlePath).To(Equal(filepath.Join(ctrBundleDir, "processes", processSpec.ID)))
		})

		It("creates a runc container based on the bundle", func() {
			Eventually(containerCreator.CreateCallCount).Should(Equal(1))
			_, actualBundlePath, actualContainerID, _ := containerCreator.CreateArgsForCall(0)
			Expect(actualBundlePath).To(Equal(filepath.Join(ctrBundleDir, "processes", processSpec.ID)))
			Expect(actualContainerID).To(Equal(processSpec.ID))
		})

		It("returns process with expected ID", func() {
			Expect(process.ID()).To(Equal(processSpec.ID))
		})

		Context("when the process spec has no ID", func() {
			BeforeEach(func() {
				processSpec.ID = ""
			})

			It("generates process ID", func() {
				processDirs, err := ioutil.ReadDir(filepath.Join(ctrBundleDir, "processes"))
				Expect(err).NotTo(HaveOccurred())
				Expect(processDirs).To(HaveLen(1))
			})
		})

		Context("when no working dir is specified", func() {
			BeforeEach(func() {
				processSpec.Dir = ""
			})

			It("defaults to /", func() {
				Expect(processBuilder.BuildProcessCallCount()).To(Equal(1))
				_, actualProcessSpec := processBuilder.BuildProcessArgsForCall(0)
				Expect(actualProcessSpec.Dir).To(Equal("/"))
			})
		})

		Context("when no user is specified", func() {
			BeforeEach(func() {
				processSpec.User = ""
			})

			It("defaults to 0:0", func() {
				Expect(processBuilder.BuildProcessCallCount()).To(Equal(1))
				_, actualProcessSpec := processBuilder.BuildProcessArgsForCall(0)
				Expect(actualProcessSpec.ContainerUID).To(Equal(0))
				Expect(actualProcessSpec.ContainerGID).To(Equal(0))
			})
		})

		Context("when limits are provided", func() {
			BeforeEach(func() {
				processSpec.OverrideContainerLimits = &garden.ProcessLimits{}
			})

			It("provides an explicit cgroup path to bundle generation", func() {
				Expect(bundleGenerator.GenerateCallCount()).To(Equal(1))
				actualCtrSpec, _ := bundleGenerator.GenerateArgsForCall(0)
				Expect(actualCtrSpec.CgroupPath).To(Equal(processSpec.ID))
			})
		})

		Context("when bind mounts are provided", func() {
			BeforeEach(func() {
				processSpec.BindMounts = []garden.BindMount{
					garden.BindMount{SrcPath: "/path/to/src", DstPath: "/path/to/dst"},
				}
			})

			It("passes the bind mounts to bundle generation", func() {
				Expect(bundleGenerator.GenerateCallCount()).To(Equal(1))
				actualCtrSpec, _ := bundleGenerator.GenerateArgsForCall(0)
				Expect(len(actualCtrSpec.BindMounts)).To(Equal(1))
				Expect(actualCtrSpec.BindMounts[0].SrcPath).To(Equal("/path/to/src"))
				Expect(actualCtrSpec.BindMounts[0].DstPath).To(Equal("/path/to/dst"))
			})
		})

		Describe("Process Wait", func() {
			var (
				exitCode int
				waitErr  error
			)

			JustBeforeEach(func() {
				exitCode, waitErr = process.Wait()
			})

			Describe("Clean up", func() {
				It("cleans up volumes", func() {
					Expect(volumizer.DestroyCallCount()).To(Equal(1))
					_, actualHandle := volumizer.DestroyArgsForCall(0)
					Expect(actualHandle).To(Equal(processSpec.ID))
				})

				It("cleans up process dir", func() {
					processPath := filepath.Join(ctrBundleDir, "processes", process.ID())
					Expect(processPath).NotTo(BeADirectory())
				})
			})

			Context("when process exits with non-zero code", func() {
				BeforeEach(func() {
					cmdThatFails := exec.Command("bash", "-c", "exit 42")
					if runtime.GOOS == "windows" {
						cmdThatFails = exec.Command("cmd", "/c", "exit 42")
					}

					realExitErr := cmdThatFails.Run()
					containerCreator.CreateReturns(logging.WrappedError{Underlying: realExitErr})
				})

				It("returns the container creation error", func() {
					Expect(waitErr).NotTo(HaveOccurred())
					Expect(exitCode).To(Equal(42))
				})
			})

			Context("when the container creation fails", func() {
				BeforeEach(func() {
					containerCreator.CreateReturns(errors.New("mango"))
				})

				It("returns the container creation error", func() {
					Expect(waitErr).To(MatchError("mango"))
				})
			})
		})

		It("creates a signaller for the process", func() {
			Expect(signallerFactory.NewSignallerCallCount()).To(Equal(1))
			Expect(signallerFactory.NewSignallerArgsForCall(0)).To(Equal(filepath.Join(ctrBundleDir, "processes", process.ID(), "pidfile")))
		})

		Describe("Process signalling", func() {
			It("uses the signaller to signal the process", func() {
				signaller.SignalReturns(errors.New("signalled"))
				Expect(process.Signal(garden.SignalKill)).To(MatchError("signalled"))
				Expect(signaller.SignalCallCount()).To(Equal(1))
				Expect(signaller.SignalArgsForCall(0)).To(Equal(garden.SignalKill))
			})
		})
	})

	Describe("pea creation failing", func() {
		var (
			createErr error
		)

		JustBeforeEach(func() {
			_, createErr = peaCreator.CreatePea(log, processSpec, garden.ProcessIO{}, ctrHandle, ctrBundleDir)
		})

		Context("when the bundle generator returns an error", func() {
			BeforeEach(func() {
				pidGetter.PidReturns(-1, errors.New("pickle"))
			})

			It("returns a wrapped error", func() {
				Expect(createErr).To(MatchError(ContainSubstring("pickle")))
			})
		})

		Context("when the bundle generator returns an error", func() {
			BeforeEach(func() {
				bundleGenerator.GenerateReturns(goci.Bndl{}, errors.New("banana"))
			})

			It("returns a wrapped error", func() {
				Expect(createErr).To(MatchError(ContainSubstring("banana")))
			})
		})

		Context("when the bundle saver returns an error", func() {
			BeforeEach(func() {
				bundleSaver.SaveReturns(errors.New("papaya"))
			})

			It("returns a wrapped error", func() {
				Expect(createErr).To(MatchError(ContainSubstring("papaya")))
			})
		})

		Context("when the volume creator returns an error", func() {
			BeforeEach(func() {
				volumizer.CreateReturns(specs.Spec{}, errors.New("coconut"))
			})

			It("returns a wrapped error", func() {
				Expect(createErr).To(MatchError(ContainSubstring("coconut")))
			})
		})

		Context("when the user is specified as a username, not a uid:gid", func() {
			BeforeEach(func() {
				processSpec.User = "frank"
			})

			It("returns an error", func() {
				Expect(createErr).To(MatchError(ContainSubstring("frank")))
			})
		})
	})
})
