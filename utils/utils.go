// Package utils contains commonly useful functions from Deisctl

package utils

import (
	_ "bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/coreos/go-etcd/etcd"
	"github.com/deis/deisctl/constant"
	"github.com/docker/docker/api/client"
	uuid "github.com/satori/go.uuid"
)

// NewUuid returns a new V4-style unique identifier.
func NewUuid() string {
	u1 := uuid.NewV4()
	s1 := fmt.Sprintf("%s", u1)
	return strings.Split(s1, "-")[0]
}

func getetcdClient() *etcd.Client {

	machines := []string{"http://127.0.0.1:4001/"}
	c := etcd.NewClient(machines)
	return c
}

func GetKey(dir, key, perm string) string {
	c := getetcdClient()
	result, err := c.Get(dir+key, false, false)
	if err != nil || result.Node.Value == "" {
		return os.Getenv(perm)
	}
	return result.Node.Value
}

func GetNewClient() (
	cli *client.DockerCli, stdout *io.PipeReader, stdoutPipe *io.PipeWriter) {
	testDaemonAddr := "/var/run/docker.sock"
	testDaemonProto := "unix"
	stdout, stdoutPipe = io.Pipe()
	cli = client.NewDockerCli(
		nil, stdoutPipe, nil, testDaemonProto, testDaemonAddr, nil)
	return
}

func PullImage(cli *client.DockerCli, args ...string) error {
	fmt.Println("pulling image : " + args[0])
	err := cli.CmdPull(args...)
	return err
}

func GetServices() []string {
	service := []string{
		"deis-builder.service",
		"deis-builder-data.service",
		"deis-cache.service",
		"deis-controller.service",
		"deis-database.service",
		"deis-database-data.service",
		"deis-logger.service",
		"deis-logger-data.service",
		"deis-registry.service",
		"deis-registry-data.service",
		"deis-router.service",
	}
	return service
}

// getClientID returns the CoreOS Machine ID or an unknown UUID string
func GetClientID() string {
	machineID := GetMachineID("/")
	if machineID == "" {
		return fmt.Sprintf("{unknown-" + NewUuid() + "}")
	}
	return machineID
}

func GetMachineID(root string) string {
	fullPath := filepath.Join(root, constant.MachineId)
	id, err := ioutil.ReadFile(fullPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(id))
}

func GetVersion() string {
	id, err := ioutil.ReadFile(constant.Version)
	if err != nil {
		return "0.0.0"
	}
	return strings.TrimSpace(string(id))
}

// GetFileBytes returns a byte array of the contents of a file.
func GetFileBytes(filename string) []byte {
	file, _ := os.Open(filename)
	defer file.Close()
	stat, _ := file.Stat()
	bs := make([]byte, stat.Size())
	_, _ = file.Read(bs)
	return bs
}

func ListFiles(dir string) ([]string, error) {
	files, err := filepath.Glob(dir)
	return files, err
}

// CreateFile creates an empty file at the specified path.
func CreateFile(path string) error {
	fo, err := os.Create(path)
	if err != nil {
		return err
	}
	defer fo.Close()
	return nil
}

func Extract(file, dir string) (err error) {
	var wd, _ = os.Getwd()
	_ = os.Chdir(dir)
	cmdl := exec.Command("tar", "-C", "/", "-xvf", file)
	if _, _, err := RunCommandWithStdoutStderr(cmdl); err != nil {
		fmt.Printf("Failed:\n%v", err)
		return err
	}
	_ = os.Chdir(wd)
	return nil
}

// GetUserDetails returns sections of a UUID.
func GetUserDetails() (string, string) {
	u1 := uuid.NewV4()
	s1 := fmt.Sprintf("%s", u1)
	return strings.Split(s1, "-")[0], strings.Split(s1, "-")[1]
}

// GetHostOs returns either "darwin" or "ubuntu".
func GetHostOs() string {
	cmd := exec.Command("uname")
	out, _ := cmd.Output()
	if strings.Contains(string(out), "Darwin") {
		return "darwin"
	}
	return "ubuntu"
}

// GetHostIPAddress returns the host IP for accessing etcd and Deis services.
func GetHostIPAddress() string {
	IP := os.Getenv("HOST_IPADDR")
	if IP == "" {
		IP = "172.17.8.100"
	}
	return IP
}

func PutVersion(version string) error {
	err := ioutil.WriteFile(constant.Version, []byte(version), 0644)
	if err != nil {
		return err
	}
	return nil
}

// Append grows a string array by appending a new element.
func Append(slice []string, data string) []string {
	m := len(slice)
	n := m + 1
	if n > cap(slice) { // if necessary, reallocate
		// allocate double what's needed, for future growth.
		newSlice := make([]string, (n + 1))
		copy(newSlice, slice)
		slice = newSlice
	}
	slice = slice[0:n]
	slice[n-1] = data
	return slice
}

// GetRandomPort returns an unused TCP listen port on the host.
func GetRandomPort() string {
	l, _ := net.Listen("tcp", "127.0.0.1:0") // listen on localhost
	defer l.Close()
	port := l.Addr()
	return strings.Split(port.String(), ":")[1]
}

func getExitCode(err error) (int, error) {
	exitCode := 0
	if exiterr, ok := err.(*exec.ExitError); ok {
		if procExit := exiterr.Sys().(syscall.WaitStatus); ok {
			return procExit.ExitStatus(), nil
		}
	}
	return exitCode, fmt.Errorf("failed to get exit code")
}

// RunCommandWithStdoutStderr execs a command and returns its output.

func RunCommandWithStdoutStderr(cmd *exec.Cmd) (bytes.Buffer, bytes.Buffer, error) {
	var stdout, stderr bytes.Buffer
	stderrPipe, err := cmd.StderrPipe()
	stdoutPipe, err := cmd.StdoutPipe()

	cmd.Env = os.Environ()
	if err != nil {
		fmt.Println("error at io pipes")
	}

	err = cmd.Start()
	if err != nil {
		fmt.Println("error at command start")
	}

	go func() {
		io.Copy(&stdout, stdoutPipe)
		fmt.Println(stdout.String())
	}()
	go func() {
		io.Copy(&stderr, stderrPipe)
		fmt.Println(stderr.String())
	}()
	time.Sleep(2000 * time.Millisecond)
	err = cmd.Wait()
	if err != nil {
		fmt.Println("error at command wait")
	}
	return stdout, stderr, err
}

func Execute(script string) error {
	cmdl := exec.Command("sh", "-c", script)
	if _, _, err := RunCommandWithStdoutStderr(cmdl); err != nil {
		fmt.Println("(Error )")
		return err
	}
	return nil
}

func logDone(message string) {
	fmt.Printf("[PASSED]: %s\n", message)
}

func stripTrailingCharacters(target string) string {
	target = strings.Trim(target, "\n")
	target = strings.Trim(target, " ")
	return target
}

func nLines(s string) int {
	return strings.Count(s, "\n")
}

//func deis(bash string , arg string ,  cmd string )
