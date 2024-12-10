package systemd_test

import (
	"fmt"

	"github.com/poly-gun/systemd"
)

func Example() {
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
}
