# `systemd` - Marshalling, Unmarshalling Library for System Daemon Service(s).

For a basic systemd service unit file, the required sections and fields are minimal:

1. **`[Unit]`**: This is a standard section available in all types of systemd units, not just services. It provides the metadata and dependency information for the unit.
2. **`[Service]`**: This section is specific to service units and defines how the service behaves and how it is executed.
3. **`[Install]`**: This section is used to define how the service should be installed (i.e., enabled or disabled). This is important for integrating the service with the system's startup sequence. However, strictly speaking, this section is not required for a service to be started manually. It's primarily used for services that should start automatically at boot time.

Here's an example of a minimalistic systemd service file (assuming it's a service named `example.service`:

```ini
[Unit]
Description=Example Service

[Service]
ExecStart=/usr/bin/exampled

[Install]
WantedBy=multi-user.target
```

## Documentation

Official `godoc` documentation (with examples) can be found at the [Package Registry](https://pkg.go.dev/github.com/poly-gun/systemd).

## Usage

###### Add Package Dependency

```bash
go get -u github.com/poly-gun/systemd
```

###### Import & Implement

`main.go`

```go
package main

import (
	"fmt"

	"github.com/poly-gun/systemd"
)

func Main() {
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
```

- Please refer to the [code examples](./example_test.go) for additional usage and implementation details.
- See https://pkg.go.dev/github.com/poly-gun/systemd for additional documentation.

## Contributions

See the [**Contributing Guide**](./CONTRIBUTING.md) for additional details on getting started.
