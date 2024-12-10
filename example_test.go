package systemd_test

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/poly-gun/systemd"
)

func Example() {
	// Establish a Daemon struct and produce its final service-file contents.
	daemon := systemd.Daemon{
		Unit: systemd.Unit{
			Description:   "Example Description of the Daemon.",
			Documentation: "https://github.com/poly-gun/steamd",
			Wants:         "network.target",
			After:         "syslog.target network-online.target",
		},
		Service: systemd.Service{
			Type:           "exec",
			ExecStart:      "/usr/bin/example-agent",
			StandardOutput: "journal",
			StandardError:  "journal",
			Environment:    "Variable1=value1,Variable2=value2",
		},
		Install: systemd.Install{
			WantedBy: "multi-user.target",
		},
		Socket: nil,
	}

	content, e := systemd.Marshal(daemon)
	if e != nil {
		panic(e)
	}

	fmt.Println(string(content))

	// Open up an existing service and unmarshal it into a Daemon instance pointer.
	file, e := os.Open("test-data/example-agent.service")
	if e != nil {
		panic(e)
	}

	defer file.Close()

	var buffer bytes.Buffer
	if _, e := io.Copy(&buffer, file); e != nil {
		panic(e)
	}

	instance, e := systemd.Unmarshal(buffer.Bytes())
	if e != nil {
		panic(e)
	}

	fmt.Println(instance)
}
