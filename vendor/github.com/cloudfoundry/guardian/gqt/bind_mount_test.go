package gqt_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"

	"code.cloudfoundry.org/garden"
	"code.cloudfoundry.org/guardian/gqt/runner"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Bind mount", func() {
	var (
		client    *runner.RunningGarden
		container garden.Container

		// container create parms
		privilegedContainer bool
		srcPath             string                 // bm: source
		dstPath             string                 // bm: destination
		bindMountMode       garden.BindMountMode   // bm: RO or RW
		bindMountOrigin     garden.BindMountOrigin // bm: Container or Host

		// pre-existing file for permissions testing
		testFileName string
	)

	BeforeEach(func() {
		privilegedContainer = false
		container = nil
		srcPath = ""
		dstPath = ""
		bindMountMode = garden.BindMountModeRO
		bindMountOrigin = garden.BindMountOriginHost
		testFileName = ""

		srcPath, testFileName = createTestHostDirAndTestFile()
		bindMountOrigin = garden.BindMountOriginHost
	})

	JustBeforeEach(func() {
		client = runner.Start(config)

		var err error
		container, err = client.Create(
			garden.ContainerSpec{
				Privileged: privilegedContainer,
				BindMounts: []garden.BindMount{{
					SrcPath: srcPath,
					DstPath: dstPath,
					Mode:    bindMountMode,
					Origin:  bindMountOrigin,
				}},
				Network: fmt.Sprintf("10.0.%d.0/24", GinkgoParallelNode()),
			})
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		unmount(srcPath)

		err := os.RemoveAll(srcPath)
		Expect(err).ToNot(HaveOccurred())

		if container != nil {
			err := client.Destroy(container.Handle())
			Expect(err).ToNot(HaveOccurred())
		}

		Expect(client.DestroyAndStop()).To(Succeed())
	})

	Context("which is read-only", func() {
		BeforeEach(func() {
			bindMountMode = garden.BindMountModeRO
			dstPath = "/home/alice/readonly"
		})

		Context("and with privileged=true", func() {
			BeforeEach(func() {
				privilegedContainer = true
			})

			It("allows all users to read files", func() {
				readProcess := containerReadFile(container, dstPath, testFileName, "alice")
				Expect(readProcess.Wait()).To(Equal(0))
			})

			It("does not allow non-root users to write files", func() {
				writeProcess := writeFile(container, dstPath, "alice")
				Expect(writeProcess.Wait()).ToNot(Equal(0))
			})

			It("allows root to read files", func() {
				readProcess := containerReadFile(container, dstPath, testFileName, "root")
				Expect(readProcess.Wait()).To(Equal(0))
			})

			It("does not allow root to write files", func() {
				writeProcess := writeFile(container, dstPath, "root")
				Expect(writeProcess.Wait()).ToNot(Equal(0))
			})

			Describe("nested-mounts", func() {
				var nestedMountpoint string

				JustBeforeEach(func() {
					nestedMountpoint = createMountPointUnder(srcPath)
				})

				AfterEach(func() {
					unmount(nestedMountpoint)
				})

				It("allows all users to read from nested bind mounts", func() {
					nestedPath := filepath.Join(dstPath, "nested-bind")
					readProcess := containerReadFile(container, nestedPath, "nested-file", "alice")
					Expect(readProcess.Wait()).To(Equal(0))
				})

				It("allows non-root to write to nested bind mounts", func() {
					nestedPath := filepath.Join(dstPath, "nested-bind")
					writeProcess := writeFile(container, nestedPath, "alice")
					Expect(writeProcess.Wait()).To(Equal(0))
				})

				It("allows root to read from nested bind mounts", func() {
					nestedPath := filepath.Join(dstPath, "nested-bind")
					readProcess := containerReadFile(container, nestedPath, "nested-file", "alice")
					Expect(readProcess.Wait()).To(Equal(0))
				})

				It("allows root to write to from nested bind mounts", func() {
					nestedPath := filepath.Join(dstPath, "nested-bind")
					writeProcess := writeFile(container, nestedPath, "root")
					Expect(writeProcess.Wait()).To(Equal(0))
				})
			})
		})

		Context("and with privileged=false", func() {
			BeforeEach(func() {
				privilegedContainer = false
			})

			It("allows all users to read files", func() {
				readProcess := containerReadFile(container, dstPath, testFileName, "alice")
				Expect(readProcess.Wait()).To(Equal(0))
			})

			It("allows non-root users to write files", func() {
				writeProcess := writeFile(container, dstPath, "alice")
				Expect(writeProcess.Wait()).ToNot(Equal(0))
			})

			It("allows root to read files", func() {
				readProcess := containerReadFile(container, dstPath, testFileName, "root")
				Expect(readProcess.Wait()).To(Equal(0))
			})

			It("does not allow root to write files", func() {
				writeProcess := writeFile(container, dstPath, "root")
				Expect(writeProcess.Wait()).ToNot(Equal(0))
			})

			Describe("nested-mounts", func() {
				var nestedMountpoint string

				JustBeforeEach(func() {
					nestedMountpoint = createMountPointUnder(srcPath)
				})

				AfterEach(func() {
					unmount(nestedMountpoint)
				})

				It("allows all users to read from nested bind mounts", func() {
					nestedPath := filepath.Join(dstPath, "nested-bind")
					readProcess := containerReadFile(container, nestedPath, "nested-file", "alice")
					Expect(readProcess.Wait()).To(Equal(0))
				})

				It("allows non-root to write to nested bind mounts", func() {
					nestedPath := filepath.Join(dstPath, "nested-bind")
					writeProcess := writeFile(container, nestedPath, "alice")
					Expect(writeProcess.Wait()).To(Equal(0))
				})

				It("allows root to read from nested bind mounts", func() {
					nestedPath := filepath.Join(dstPath, "nested-bind")
					readProcess := containerReadFile(container, nestedPath, "nested-file", "alice")
					Expect(readProcess.Wait()).To(Equal(0))
				})

				It("allows root to write to from nested bind mounts", func() {
					nestedPath := filepath.Join(dstPath, "nested-bind")
					writeProcess := writeFile(container, nestedPath, "root")
					Expect(writeProcess.Wait()).To(Equal(0))
				})
			})
		})
	})

	Context("which is read-write", func() {
		BeforeEach(func() {
			bindMountMode = garden.BindMountModeRW
			dstPath = "/home/alice/readwrite"
		})

		Context("and with privileged=true", func() {
			BeforeEach(func() {
				privilegedContainer = true
			})

			It("allows all users to read files", func() {
				readProcess := containerReadFile(container, dstPath, testFileName, "alice")
				Expect(readProcess.Wait()).To(Equal(0))
			})

			It("does not allow non-root users to write files (since the mounted directory is owned by host-root)", func() {
				writeProcess := writeFile(container, dstPath, "alice")
				Expect(writeProcess.Wait()).ToNot(Equal(0))
			})

			It("allows root to read files", func() {
				readProcess := containerReadFile(container, dstPath, testFileName, "root")
				Expect(readProcess.Wait()).To(Equal(0))
			})

			It("allows root to write files (as container and host root are the same)", func() {
				writeProcess := writeFile(container, dstPath, "root")
				Expect(writeProcess.Wait()).To(Equal(0))
			})

			Describe("nested-mounts", func() {
				var nestedMountpoint string

				JustBeforeEach(func() {
					nestedMountpoint = createMountPointUnder(srcPath)
				})

				AfterEach(func() {
					unmount(nestedMountpoint)
				})

				It("allows all users to read from nested bind mounts", func() {
					nestedPath := filepath.Join(dstPath, "nested-bind")
					readProcess := containerReadFile(container, nestedPath, "nested-file", "alice")
					Expect(readProcess.Wait()).To(Equal(0))
				})

				It("allows non-root to write to nested bind mounts", func() {
					nestedPath := filepath.Join(dstPath, "nested-bind")
					writeProcess := writeFile(container, nestedPath, "alice")
					Expect(writeProcess.Wait()).To(Equal(0))
				})

				It("allows root to read from nested bind mounts", func() {
					nestedPath := filepath.Join(dstPath, "nested-bind")
					readProcess := containerReadFile(container, nestedPath, "nested-file", "alice")
					Expect(readProcess.Wait()).To(Equal(0))
				})

				It("allows root to write to from nested bind mounts", func() {
					nestedPath := filepath.Join(dstPath, "nested-bind")
					writeProcess := writeFile(container, nestedPath, "root")
					Expect(writeProcess.Wait()).To(Equal(0))
				})
			})
		})

		Context("and with privileged=false", func() {
			BeforeEach(func() {
				privilegedContainer = false
			})

			It("allows all users to read files", func() {
				readProcess := containerReadFile(container, dstPath, testFileName, "alice")
				Expect(readProcess.Wait()).To(Equal(0))
			})

			// the mounted directory is owned by host-root, so alice shouldnt be able to write
			It("does not allow non-root users to write files", func() {
				writeProcess := writeFile(container, dstPath, "alice")
				Expect(writeProcess.Wait()).ToNot(Equal(0))
			})

			It("allows root to read files", func() {
				readProcess := containerReadFile(container, dstPath, testFileName, "root")
				Expect(readProcess.Wait()).To(Equal(0))
			})

			// container and host root are not the same, and the mounted directory is
			// owned by host-root, so writes should fail.
			It("does not allow root to write files", func() {
				writeProcess := writeFile(container, dstPath, "root")
				Expect(writeProcess.Wait()).NotTo(Equal(0))
			})

			Describe("nested-mounts", func() {
				var nestedMountpoint string

				JustBeforeEach(func() {
					nestedMountpoint = createMountPointUnder(srcPath)
				})

				AfterEach(func() {
					unmount(nestedMountpoint)
				})

				It("allows all users to read from nested bind mounts", func() {
					nestedPath := filepath.Join(dstPath, "nested-bind")
					readProcess := containerReadFile(container, nestedPath, "nested-file", "alice")
					Expect(readProcess.Wait()).To(Equal(0))
				})

				It("allows non-root to write to nested bind mounts", func() {
					nestedPath := filepath.Join(dstPath, "nested-bind")
					writeProcess := writeFile(container, nestedPath, "alice")
					Expect(writeProcess.Wait()).To(Equal(0))
				})

				It("allows root to read from nested bind mounts", func() {
					nestedPath := filepath.Join(dstPath, "nested-bind")
					readProcess := containerReadFile(container, nestedPath, "nested-file", "alice")
					Expect(readProcess.Wait()).To(Equal(0))
				})

				It("allows root to write to from nested bind mounts", func() {
					nestedPath := filepath.Join(dstPath, "nested-bind")
					writeProcess := writeFile(container, nestedPath, "root")
					Expect(writeProcess.Wait()).To(Equal(0))
				})
			})
		})
	})
})

func createTestHostDirAndTestFile() (string, string) {
	tstHostDir, err := ioutil.TempDir("", "bind-mount-test-dir")
	Expect(err).ToNot(HaveOccurred())
	err = os.Chown(tstHostDir, 0, 0)
	Expect(err).ToNot(HaveOccurred())
	err = os.Chmod(tstHostDir, 0755)
	Expect(err).ToNot(HaveOccurred())

	var cmd *exec.Cmd
	cmd = exec.Command("mount", "--bind", tstHostDir, tstHostDir)
	Expect(cmd.Run()).To(Succeed())

	debugOut, err := exec.Command("mount").CombinedOutput()
	Expect(err).ToNot(HaveOccurred())

	cmd = exec.Command("mount", "--make-shared", tstHostDir)
	out, err := cmd.CombinedOutput()
	debugMsg := fmt.Sprintf("Command: mount --make-share %s\n%s\nOutput of `mount` command:\n%v",
		tstHostDir, string(debugOut), string(out))
	Expect(err).ToNot(HaveOccurred(), debugMsg)

	fileName := fmt.Sprintf("bind-mount-%d-test-file", GinkgoParallelNode())
	file, err := os.OpenFile(filepath.Join(tstHostDir, fileName), os.O_CREATE|os.O_RDWR, 0777)
	Expect(err).ToNot(HaveOccurred())
	Expect(file.Close()).ToNot(HaveOccurred())

	return tstHostDir, fileName
}

func createMountPointUnder(srcPath string) string {
	nestedBindPath := filepath.Join(srcPath, "nested-bind")
	Expect(os.MkdirAll(nestedBindPath, os.FileMode(0755))).To(Succeed())

	cmd := exec.Command("mount", "-t", "tmpfs", "tmpfs", nestedBindPath)
	Expect(cmd.Run()).To(Succeed())

	file, err := os.OpenFile(filepath.Join(nestedBindPath, "nested-file"), os.O_CREATE|os.O_RDWR, 0777)
	Expect(err).ToNot(HaveOccurred())
	Expect(file.Close()).ToNot(HaveOccurred())

	return nestedBindPath
}

func unmount(mountpoint string) {
	cmd := exec.Command("umount", "-f", mountpoint)
	output, err := cmd.CombinedOutput()
	if len(output) > 0 {
		fmt.Printf("Command: umount -f [%s]\n%v", mountpoint, string(output))
	}
	Expect(err).NotTo(HaveOccurred())
}

func containerReadFile(container garden.Container, dstPath, fileName, user string) garden.Process {
	filePath := filepath.Join(dstPath, fileName)

	process, err := container.Run(garden.ProcessSpec{
		Path: "cat",
		Args: []string{filePath},
		User: user,
	}, garden.ProcessIO{})
	Expect(err).ToNot(HaveOccurred())

	return process
}

func writeFile(container garden.Container, dstPath, user string) garden.Process {
	// try to write a new file
	filePath := filepath.Join(dstPath, "checkFileAccess-file")

	process, err := container.Run(garden.ProcessSpec{
		Path: "touch",
		Args: []string{filePath},
		User: user,
	}, garden.ProcessIO{
		Stderr: GinkgoWriter,
		Stdout: GinkgoWriter,
	})
	Expect(err).ToNot(HaveOccurred())

	return process
}
