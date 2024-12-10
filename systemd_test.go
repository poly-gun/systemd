package systemd_test

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/poly-gun/systemd"
)

// [Unit]
// Description=Dedicated Server
//
// After=docker.service
// Requires=docker.service
// PartOf=docker.service
//
// [Service]
// User=steam
// Group=steam
// Type=oneshot
// RemainAfterExit=true
//
// WorkingDirectory=/home/steam/.configuration
//
// ExecStartPre=/usr/bin/docker compose pull
// ExecStart=/usr/bin/docker compose up --detach --remove-orphans
// ExecStop=/usr/bin/docker compose down
//
// StandardOutput=journal
// StandardError=journal
//
// [Install]
// WantedBy=multi-user.target docker.service
func Test(t *testing.T) {
	t.Run("Docker-Marshal-Test", func(t *testing.T) {
		daemon := systemd.Daemon{
			Unit: systemd.Unit{
				Description: "Dedicated Server",
				After:       "docker.service",
				Requires:    "docker.service",
				PartOf:      "docker.service",
			},
			Service: systemd.Service{
				User:             "steam",
				Group:            "steam",
				Type:             "oneshot",
				RemainAfterExit:  "yes",
				WorkingDirectory: "/home/steam/.configuration",
				ExecStartPre:     "/usr/bin/docker compose pull",
				ExecStart:        "/usr/bin/docker compose up --detach --remove-orphans",
				ExecStop:         "/usr/bin/docker compose down",
				StandardOutput:   "journal",
				StandardError:    "journal",
			},
			Install: systemd.Install{
				WantedBy: "multi-user.target docker.service",
			},
		}

		content, e := systemd.Marshal(daemon)
		if e != nil {
			t.Errorf("fialed marshalling daemon: %v", e)
		}

		t.Logf("Daemon:\n%s", string(content))
	})

	t.Run("Agent-Marshal-Test", func(t *testing.T) {
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
			t.Errorf("fialed marshalling daemon: %v", e)
		}

		t.Logf("Daemon:\n%s", string(content))
	})

	t.Run("Agent-Unmarshal-Test", func(t *testing.T) {
		handle, e := os.Open("test-data/example-agent.service")
		if e != nil {
			t.Errorf("Failed opening agent file: %v", e)
		}

		var buffer bytes.Buffer
		if _, e := io.Copy(&buffer, handle); e != nil {
			t.Errorf("Failed reading agent file into buffer: %v", e)
		}

		instance, e := systemd.Unmarshal(buffer.Bytes())
		if e != nil {
			t.Errorf("fialed marshalling daemon: %v", e)
		}

		t.Logf("Daemon:\n%+v", instance)
	})
}
