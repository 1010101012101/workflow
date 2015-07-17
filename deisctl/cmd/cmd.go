package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/deis/deis/deisctl/backend"
	"github.com/deis/deis/deisctl/config"
	"github.com/deis/deis/deisctl/units"
	"github.com/deis/deis/deisctl/utils"
)

const (
	// PlatformCommand is shorthand for "all the Deis components."
	PlatformCommand string = "platform"
	// StatelessPlatformCommand is shorthand for the components except store-*, database, and logger.
	StatelessPlatformCommand string = "stateless-platform"
	swarm                    string = "swarm"
)

// ListUnits prints a list of installed units.
func ListUnits(b backend.Backend) error {
	return b.ListUnits()
}

// ListUnitFiles prints the contents of all defined unit files.
func ListUnitFiles(b backend.Backend) error {
	return b.ListUnitFiles()
}

// Location to write standard output. By default, this is the os.Stdout.
var Stdout io.Writer = os.Stderr

// Location to write standard error information. By default, this is the os.Stderr.
var Stderr io.Writer = os.Stdout

// Scale grows or shrinks the number of running components.
// Currently "router", "registry" and "store-gateway" are the only types that can be scaled.
func Scale(targets []string, b backend.Backend) error {
	var wg sync.WaitGroup

	for _, target := range targets {
		component, num, err := splitScaleTarget(target)
		if err != nil {
			return err
		}
		// the router, registry, and store-gateway are the only component that can scale at the moment
		if !strings.Contains(component, "router") && !strings.Contains(component, "registry") && !strings.Contains(component, "store-gateway") {
			return fmt.Errorf("cannot scale %s component", component)
		}
		b.Scale(component, num, &wg, Stdout, Stderr)
		wg.Wait()
	}
	return nil
}

// Start activates the specified components.
func Start(targets []string, b backend.Backend) error {

	// if target is platform, install all services
	if len(targets) == 1 {
		if targets[0] == PlatformCommand {
			return StartPlatform(b, false)
		} else if targets[0] == StatelessPlatformCommand {
			return StartPlatform(b, true)
		} else if targets[0] == swarm {
			return StartSwarm(b)
		}
	}
	var wg sync.WaitGroup

	b.Start(targets, &wg, Stdout, Stderr)
	wg.Wait()

	return nil
}

// CheckRequiredKeys exist in etcd
func CheckRequiredKeys() error {
	if err := config.CheckConfig("/deis/platform/", "domain"); err != nil {
		return fmt.Errorf(`Missing platform domain, use:
deisctl config platform set domain=<your-domain>`)
	}

	if err := config.CheckConfig("/deis/platform/", "sshPrivateKey"); err != nil {
		fmt.Printf(`Warning: Missing sshPrivateKey, "deis run" will be unavailable. Use:
deisctl config platform set sshPrivateKey=<path-to-key>
`)
	}
	return nil
}

func startDefaultServices(b backend.Backend, stateless bool, wg *sync.WaitGroup, out, err io.Writer) {

	// Wait for groups to come up.
	// If we're running in stateless mode, we start only a subset of services.
	if !stateless {
		fmt.Fprintln(out, "Storage subsystem...")
		b.Start([]string{"store-monitor"}, wg, out, err)
		wg.Wait()
		b.Start([]string{"store-daemon"}, wg, out, err)
		wg.Wait()
		b.Start([]string{"store-metadata"}, wg, out, err)
		wg.Wait()

		// we start gateway first to give metadata time to come up for volume
		b.Start([]string{"store-gateway@*"}, wg, out, err)
		wg.Wait()
		b.Start([]string{"store-volume"}, wg, out, err)
		wg.Wait()
	}

	// start logging subsystem first to collect logs from other components
	fmt.Fprintln(out, "Logging subsystem...")
	if !stateless {
		b.Start([]string{"logger"}, wg, out, err)
		wg.Wait()
	}
	b.Start([]string{"logspout"}, wg, out, err)
	wg.Wait()

	// Start these in parallel. This section can probably be removed now.
	var bgwg sync.WaitGroup
	var trash bytes.Buffer
	batch := []string{
		"database", "registry@*", "controller", "builder",
		"publisher", "router@*",
	}
	if stateless {
		batch = []string{"registry@*", "controller", "builder", "publisher", "router@*"}
	}
	b.Start(batch, &bgwg, &trash, &trash)
	// End background stuff.

	fmt.Fprintln(Stdout, "Control plane...")
	batch = []string{"database", "registry@*", "controller"}
	if stateless {
		batch = []string{"registry@*", "controller"}
	}
	b.Start(batch, wg, out, err)
	wg.Wait()

	b.Start([]string{"builder"}, wg, out, err)
	wg.Wait()

	fmt.Fprintln(out, "Data plane...")
	b.Start([]string{"publisher"}, wg, out, err)
	wg.Wait()

	fmt.Fprintln(out, "Routing mesh...")
	b.Start([]string{"router@*"}, wg, out, err)
	wg.Wait()
}

// Stop deactivates the specified components.
func Stop(targets []string, b backend.Backend) error {

	// if target is platform, stop all services
	if len(targets) == 1 {
		if targets[0] == PlatformCommand {
			return StopPlatform(b, false)
		} else if targets[0] == StatelessPlatformCommand {
			return StopPlatform(b, true)
		} else if targets[0] == swarm {
			return StopSwarm(b)
		}
	}

	var wg sync.WaitGroup

	b.Stop(targets, &wg, Stdout, Stderr)
	wg.Wait()

	return nil
}

func stopDefaultServices(b backend.Backend, stateless bool, wg *sync.WaitGroup, out, err io.Writer) {

	fmt.Fprintln(out, "Routing mesh...")
	b.Stop([]string{"router@*"}, wg, out, err)
	wg.Wait()

	fmt.Fprintln(out, "Data plane...")
	b.Stop([]string{"publisher"}, wg, out, err)
	wg.Wait()

	fmt.Fprintln(out, "Control plane...")
	if stateless {
		b.Stop([]string{"controller", "builder", "registry@*"}, wg, out, err)
	} else {
		b.Stop([]string{"controller", "builder", "database", "registry@*"}, wg, out, err)
	}
	wg.Wait()

	fmt.Fprintln(out, "Logging subsystem...")
	if stateless {
		b.Stop([]string{"logspout"}, wg, out, err)
	} else {
		b.Stop([]string{"logger", "logspout"}, wg, out, err)
	}
	wg.Wait()

	if !stateless {
		fmt.Fprintln(out, "Storage subsystem...")
		b.Stop([]string{"store-volume", "store-gateway@*"}, wg, out, err)
		wg.Wait()
		b.Stop([]string{"store-metadata"}, wg, out, err)
		wg.Wait()
		b.Stop([]string{"store-daemon"}, wg, out, err)
		wg.Wait()
		b.Stop([]string{"store-monitor"}, wg, out, err)
		wg.Wait()
	}

}

// Restart stops and then starts the specified components.
func Restart(targets []string, b backend.Backend) error {

	// act as if the user called "stop" and then "start"
	if err := Stop(targets, b); err != nil {
		return err
	}

	return Start(targets, b)
}

// Status prints the current status of components.
func Status(targets []string, b backend.Backend) error {

	for _, target := range targets {
		if err := b.Status(target); err != nil {
			return err
		}
	}
	return nil
}

// Journal prints log output for the specified components.
func Journal(targets []string, b backend.Backend) error {

	for _, target := range targets {
		if err := b.Journal(target); err != nil {
			return err
		}
	}
	return nil
}

// Install loads the definitions of components from local unit files.
// After Install, the components will be available to Start.
func Install(targets []string, b backend.Backend, checkKeys func() error) error {

	// if target is platform, install all services
	if len(targets) == 1 {
		if targets[0] == PlatformCommand {
			return InstallPlatform(b, checkKeys, false)
		} else if targets[0] == StatelessPlatformCommand {
			return InstallPlatform(b, checkKeys, true)
		} else if targets[0] == swarm {
			return InstallSwarm(b)
		}
	}
	var wg sync.WaitGroup

	// otherwise create the specific targets
	b.Create(targets, &wg, Stdout, Stderr)
	wg.Wait()

	return nil
}

func installDefaultServices(b backend.Backend, stateless bool, wg *sync.WaitGroup, out, err io.Writer) {

	if !stateless {
		fmt.Fprintln(out, "Storage subsystem...")
		b.Create([]string{"store-daemon", "store-monitor", "store-metadata", "store-volume", "store-gateway@1"}, wg, out, err)
		wg.Wait()
	}

	fmt.Fprintln(out, "Logging subsystem...")
	if stateless {
		b.Create([]string{"logspout"}, wg, out, err)
	} else {
		b.Create([]string{"logger", "logspout"}, wg, out, err)
	}
	wg.Wait()

	fmt.Fprintln(out, "Control plane...")
	if stateless {
		b.Create([]string{"registry@1", "controller", "builder"}, wg, out, err)
	} else {
		b.Create([]string{"database", "registry@1", "controller", "builder"}, wg, out, err)
	}
	wg.Wait()

	fmt.Fprintln(out, "Data plane...")
	b.Create([]string{"publisher"}, wg, out, err)
	wg.Wait()

	fmt.Fprintln(out, "Routing mesh...")
	b.Create([]string{"router@1", "router@2", "router@3"}, wg, out, err)
	wg.Wait()

}

// Uninstall unloads the definitions of the specified components.
// After Uninstall, the components will be unavailable until Install is called.
func Uninstall(targets []string, b backend.Backend) error {
	if len(targets) == 1 {
		if targets[0] == PlatformCommand {
			return UninstallPlatform(b, false)
		} else if targets[0] == StatelessPlatformCommand {
			return UninstallPlatform(b, true)
		} else if targets[0] == swarm {
			return UnInstallSwarm(b)
		}
	}

	var wg sync.WaitGroup

	// uninstall the specific target
	b.Destroy(targets, &wg, Stdout, Stderr)
	wg.Wait()

	return nil
}

func uninstallAllServices(b backend.Backend, stateless bool, wg *sync.WaitGroup, out, err io.Writer) error {

	fmt.Fprintln(out, "Routing mesh...")
	b.Destroy([]string{"router@*"}, wg, out, err)
	wg.Wait()

	fmt.Fprintln(out, "Data plane...")
	b.Destroy([]string{"publisher"}, wg, out, err)
	wg.Wait()

	fmt.Fprintln(out, "Control plane...")
	if stateless {
		b.Destroy([]string{"controller", "builder", "registry@*"}, wg, out, err)
	} else {
		b.Destroy([]string{"controller", "builder", "database", "registry@*"}, wg, out, err)
	}
	wg.Wait()

	fmt.Fprintln(out, "Logging subsystem...")
	if stateless {
		b.Destroy([]string{"logspout"}, wg, out, err)
	} else {
		b.Destroy([]string{"logger", "logspout"}, wg, out, err)
	}
	wg.Wait()

	if !stateless {
		fmt.Fprintln(out, "Storage subsystem...")
		b.Destroy([]string{"store-volume", "store-gateway@*"}, wg, out, err)
		wg.Wait()
		b.Destroy([]string{"store-metadata"}, wg, out, err)
		wg.Wait()
		b.Destroy([]string{"store-daemon"}, wg, out, err)
		wg.Wait()
		b.Destroy([]string{"store-monitor"}, wg, out, err)
		wg.Wait()
	}

	return nil
}

func splitScaleTarget(target string) (c string, num int, err error) {
	r := regexp.MustCompile(`([a-z-]+)=([\d]+)`)
	match := r.FindStringSubmatch(target)
	if len(match) == 0 {
		err = fmt.Errorf("Could not parse: %v", target)
		return
	}
	c = match[1]
	num, err = strconv.Atoi(match[2])
	if err != nil {
		return
	}
	return
}

// Config gets or sets a configuration value from the cluster.
//
// A configuration value is stored and retrieved from a key/value store (in this case, etcd)
// at /deis/<component>/<config>. Configuration values are typically used for component-level
// configuration, such as enabling TLS for the routers.
func Config(target string, action string, key []string) error {
	if err := config.Config(target, action, key); err != nil {
		return err
	}
	return nil
}

// RefreshUnits overwrites local unit files with those requested.
// Downloading from the Deis project GitHub URL by tag or SHA is the only mechanism
// currently supported.
func RefreshUnits(dir, tag, url string) error {
	dir = utils.ResolvePath(dir)
	// create the target dir if necessary
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	// download and save the unit files to the specified path
	for _, unit := range units.Names {
		src := fmt.Sprintf(url, tag, unit)
		dest := filepath.Join(dir, unit+".service")
		res, err := http.Get(src)
		if err != nil {
			return err
		}
		if res.StatusCode != 200 {
			return errors.New(res.Status)
		}
		defer res.Body.Close()
		data, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		if err = ioutil.WriteFile(dest, data, 0644); err != nil {
			return err
		}
		fmt.Printf("Refreshed %s from %s\n", unit, tag)
	}
	return nil
}

// SSH opens an interactive shell on a machine in the cluster
func SSH(target string, b backend.Backend) error {
	if err := b.SSH(target); err != nil {
		return err
	}

	return nil
}
