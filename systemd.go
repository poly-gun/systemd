package systemd

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strings"

	"gopkg.in/ini.v1"
)

type tag struct {
	Name     string
	Field    string
	Optional bool

	Value *string
}

// Unit represents the [Unit] section of a systemd service file.
//
// The [Unit] section of a systemd service file is used to specify metadata and dependencies of the unit. This section is the starting point for unit
// configuration and plays a crucial role in how systemd understands the service. Below is a comprehensive list of options that can be used within the `[Unit]`
// section, along with their descriptions:
//
// These options provide a comprehensive toolkit for configuring how your service interacts with the rest of the system and its units, allowing for precise
// control over service behavior, dependencies, and lifecycle.
type Unit struct {
	Description           string `json:"Description" yaml:"Description" ini:"Description" systemd:"Description"`                                                                                 // Provides a brief explanation of the unit and its functionality.
	Documentation         string `json:"Documentation,omitempty" yaml:"Documentation,omitempty" ini:"Documentation,omitempty" systemd:"Documentation,omitempty"`                                 // Provides a list of URIs referencing documentation for the unit.
	Requires              string `json:"Requires,omitempty" yaml:"Requires,omitempty" ini:"Requires,omitempty" systemd:"Requires,omitempty"`                                                     // Configures dependency units, which must be started along with the unit.
	Requisite             string `json:"Requisite,omitempty" yaml:"Requisite,omitempty" ini:"Requisite,omitempty" systemd:"Requisite,omitempty"`                                                 // Similar to Requires, but if the units are not started already, the unit itself will fail to start.
	Wants                 string `json:"Wants,omitempty" yaml:"Wants,omitempty" ini:"Wants,omitempty" systemd:"Wants,omitempty"`                                                                 // A weaker version of Requires. If the units listed are not found, the unit will continue to start.
	BindsTo               string `json:"BindsTo,omitempty" yaml:"BindsTo,omitempty" ini:"BindsTo,omitempty" systemd:"BindsTo,omitempty"`                                                         // Stronger than Requires. If the units listed stop, this unit will also stop.
	PartOf                string `json:"PartOf,omitempty" yaml:"PartOf,omitempty" ini:"PartOf,omitempty" systemd:"PartOf,omitempty"`                                                             // If the units listed are stopped or restarted, this unit will stop or restart too.
	Conflicts             string `json:"Conflicts,omitempty" yaml:"Conflicts,omitempty" ini:"Conflicts,omitempty" systemd:"Conflicts,omitempty"`                                                 // Specifies units that cannot be run simultaneously with this unit. If both units are started, the conflicting unit will be stopped.
	Before                string `json:"Before,omitempty" yaml:"Before,omitempty" ini:"Before,omitempty" systemd:"Before,omitempty"`                                                             // Indicates that the unit should be started before the units listed.
	After                 string `json:"After,omitempty" yaml:"After,omitempty" ini:"After,omitempty" systemd:"After,omitempty"`                                                                 // Indicates that the unit should be started after the units listed.
	OnFailure             string `json:"OnFailure,omitempty" yaml:"OnFailure,omitempty" ini:"OnFailure,omitempty" systemd:"OnFailure,omitempty"`                                                 // Specifies units to be activated when this unit fails.
	PropagatesReloadTo    string `json:"PropagatesReloadTo,omitempty" yaml:"PropagatesReloadTo,omitempty" ini:"PropagatesReloadTo,omitempty" systemd:"PropagatesReloadTo,omitempty"`             // Units listed will be reloaded when this unit is reloaded.
	ReloadPropagatedFrom  string `json:"ReloadPropagatedFrom,omitempty" yaml:"ReloadPropagatedFrom,omitempty" ini:"ReloadPropagatedFrom,omitempty" systemd:"ReloadPropagatedFrom,omitempty"`     // Opposite of PropagatesReloadTo. This unit will be reloaded when the units listed are reloaded.
	JoinsNamespaceOf      string `json:"JoinsNamespaceOf,omitempty" yaml:"JoinsNamespaceOf,omitempty" ini:"JoinsNamespaceOf,omitempty" systemd:"JoinsNamespaceOf,omitempty"`                     // Specifies that this unit will join the namespace of the units listed.
	RequiresMountsFor     string `json:"RequiresMountsFor,omitempty" yaml:"RequiresMountsFor,omitempty" ini:"RequiresMountsFor,omitempty" systemd:"RequiresMountsFor,omitempty"`                 // Automatically adds dependencies of type Requires= and After= for all mount units required to access the specified path.
	OnFailureJobMode      string `json:"OnFailureJobMode,omitempty" yaml:"OnFailureJobMode,omitempty" ini:"OnFailureJobMode,omitempty" systemd:"OnFailureJobMode,omitempty"`                     // Configures the job mode to apply to the units listed in OnFailure=.
	IgnoreOnIsolate       string `json:"IgnoreOnIsolate,omitempty" yaml:"IgnoreOnIsolate,omitempty" ini:"IgnoreOnIsolate,omitempty" systemd:"IgnoreOnIsolate,omitempty"`                         // If set to true, isolating this unit will not affect the unit. It is mainly used with target units.
	StopWhenUnneeded      string `json:"StopWhenUnneeded,omitempty" yaml:"StopWhenUnneeded,omitempty" ini:"StopWhenUnneeded,omitempty" systemd:"StopWhenUnneeded,omitempty"`                     // If true, this unit will be stopped when it is no longer used.
	RefuseManualStart     string `json:"RefuseManualStart,omitempty" yaml:"RefuseManualStart,omitempty" ini:"RefuseManualStart,omitempty" systemd:"RefuseManualStart,omitempty"`                 // If set to yes, this unit cannot be started manually.
	RefuseManualStop      string `json:"RefuseManualStop,omitempty" yaml:"RefuseManualStop,omitempty" ini:"RefuseManualStop,omitempty" systemd:"RefuseManualStop,omitempty"`                     // Similar to RefuseManualStart, but prevents the unit from being stopped manually.
	AllowIsolate          string `json:"AllowIsolate,omitempty" yaml:"AllowIsolate,omitempty" ini:"AllowIsolate,omitempty" systemd:"AllowIsolate,omitempty"`                                     // Allows or disallows the unit to be isolated from other units.
	DefaultDependencies   string `json:"DefaultDependencies,omitempty" yaml:"DefaultDependencies,omitempty" ini:"DefaultDependencies,omitempty" systemd:"DefaultDependencies,omitempty"`         // Specifies whether or not default dependencies (Requires= and After= for basic.target and Conflicts= and Before= for shutdown.target) are added.
	JobTimeoutSec         string `json:"JobTimeoutSec,omitempty" yaml:"JobTimeoutSec,omitempty" ini:"JobTimeoutSec,omitempty" systemd:"JobTimeoutSec,omitempty"`                                 // Specifies the time to wait for the job to complete. A job is the operation of starting or stopping the unit.
	JobTimeoutAction      string `json:"JobTimeoutAction,omitempty" yaml:"JobTimeoutAction,omitempty" ini:"JobTimeoutAction,omitempty" systemd:"JobTimeoutAction,omitempty"`                     // **JobTimeoutRebootArgument**: Specify the action to take if the job timeout is reached.
	StartLimitIntervalSec string `json:"StartLimitIntervalSec,omitempty" yaml:"StartLimitIntervalSec,omitempty" ini:"StartLimitIntervalSec,omitempty" systemd:"StartLimitIntervalSec,omitempty"` // **StartLimitBurst**: These options are used to configure rate limiting for the start operation of the unit.
	StartLimitAction      string `json:"StartLimitAction,omitempty" yaml:"StartLimitAction,omitempty" ini:"StartLimitAction,omitempty" systemd:"StartLimitAction,omitempty"`                     // Determines the action to take if the rate limit specified by the previous options is exceeded.
	Condition             string `json:"Condition,omitempty" yaml:"Condition,omitempty" ini:"Condition,omitempty" systemd:"Condition,omitempty"`                                                 // Allows specifying a condition that must be met for the unit to be started.
	Assert                string `json:"Assert,omitempty" yaml:"Assert,omitempty" ini:"Assert,omitempty" systemd:"Assert,omitempty"`                                                             // Similar to Condition, but if the condition is not met, the unit will be considered failed.
	SourcePath            string `json:"SourcePath,omitempty" yaml:"SourcePath,omitempty" ini:"SourcePath,omitempty" systemd:"SourcePath,omitempty"`                                             // Specifies the source configuration file path of the unit.
}

func (u *Unit) tags() (tags []*tag) {
	const key = "systemd"

	tags = make([]*tag, 0)
	instance := reflect.ValueOf(u).Elem()
	structure := reflect.TypeOf(u).Elem()

	for i := 0; i < structure.NumField(); i++ {
		field := structure.Field(i)
		if v, ok := field.Tag.Lookup(key); ok {
			partials := strings.Split(v, ",")
			for idx, partial := range partials {
				partials[idx] = strings.TrimSpace(partial)
			}

			var optional bool
			if len(partials) > 1 {
				for _, partial := range partials {
					partial = strings.ToLower(partial)
					if partial == "omitempty" {
						optional = true
					}
				}
			}

			var assignment *string
			if value := instance.Field(i).String(); value != "" {
				assignment = &value
			}

			attribute := &tag{
				Name:     partials[0],
				Field:    field.Name,
				Optional: optional,
				Value:    assignment,
			}

			tags = append(tags, attribute)
		}
	}

	return
}

// assignments represents the set of key-values to write to a systemd file.
func (u *Unit) assignments() (exports map[string]string) {
	exports = make(map[string]string)

	for _, tag := range u.tags() {
		if !(tag.Optional) || tag.Value != nil {
			exports[tag.Name] = *tag.Value
		}
	}

	return
}

// Export represents the service's raw buffer of a [Unit] section.
func (u *Unit) export() (*bytes.Buffer, error) {
	file := ini.Empty(ini.LoadOptions{
		// Loose:                       false,
		// Insensitive:                 false,
		// InsensitiveSections:         false,
		// InsensitiveKeys:             false,
		// IgnoreContinuation:          false,
		// IgnoreInlineComment:         false,
		// SkipUnrecognizableLines:     false,
		// ShortCircuit:                false,
		// AllowBooleanKeys:            false,
		// AllowShadows:                false,
		// AllowNestedValues:           false,
		// AllowPythonMultilineValues:  false,
		// SpaceBeforeInlineComment:    false,
		// UnescapeValueDoubleQuotes:   false,
		// UnescapeValueCommentSymbols: false,
		// UnparseableSections:         nil,
		// KeyValueDelimiters:          "",
		// KeyValueDelimiterOnWrite:    "",
		// ChildSectionDelimiter:       "",
		// PreserveSurroundedQuote:     false,
		// DebugFunc:                   nil,
		// ReaderBufferSize:            0,
		// AllowNonUniqueSections:      false,
		// AllowDuplicateShadowValues:  false,
	})

	section, e := file.NewSection("Unit")
	if e != nil {
		return nil, e
	}

	for key, value := range u.assignments() {
		if _, e := section.NewKey(key, value); e != nil {
			return nil, e
		}
	}

	var buffer, output bytes.Buffer
	if _, e := file.WriteTo(&buffer); e != nil {
		return nil, e
	}

	scanner := bufio.NewScanner(&buffer)
	for scanner.Scan() {
		line := scanner.Bytes()
		if key, value, valid := bytes.Cut(line, []byte("=")); valid {
			var array = [][]byte{bytes.TrimSpace(key), bytes.TrimSpace(value)}

			line = append(bytes.Join(array, []byte("=")), []byte("\n")...)
			if _, e := output.Write(line); e != nil {
				return nil, e
			}

			continue
		}

		output.Write(append(line, []byte("\n")...))
	}

	if e := scanner.Err(); e != nil {
		return nil, e
	}

	if _, e := output.Write([]byte("\n")); e != nil {
		return nil, e
	}

	return &output, nil
}

// Service represents the [Service] section of a systemd service file.
//
// The [Service] section of a systemd service file specifies how the service should be started and how it behaves at runtime. Here is a comprehensive list of
// the options you can use within the `[Service]` section, along with their descriptions:
//
// These options allow you to control the execution environment, resource utilization, and security policies for your systemd services. The right combination of these settings depends on the specific needs of your service and the security requirements of your system. Always consult the latest systemd documentation for the most comprehensive and detailed descriptions of these options, as there are often new settings and changes with each systemd release.
type Service struct {
	Type                     string `json:"Type,omitempty" yaml:"Type,omitempty" ini:"Type,omitempty" systemd:"Type,omitempty"`                                                                                 // Specifies the type of the service. Common values include `simple`, `forking`, `oneshot`, `dbus`, `notify`, and `idle`. Defaults to "simple".
	ExecStart                string `json:"ExecStart" yaml:"ExecStart" ini:"ExecStart" systemd:"ExecStart"`                                                                                                     // Commands or script that are executed when the service is started. This is the main command for the service. See related (ExecStart, ExecStartPre, ExecStartPost, ExecStop, ExecReload)
	ExecStartPre             string `json:"ExecStartPre,omitempty" yaml:"ExecStartPre,omitempty" ini:"ExecStartPre,omitempty" systemd:"ExecStartPre,omitempty"`                                                 // Commands or scripts that are executed before ExecStart. See related (ExecStart, ExecStartPre, ExecStartPost, ExecStop, ExecReload)
	ExecStartPost            string `json:"ExecStartPost,omitempty" yaml:"ExecStartPost,omitempty" ini:"ExecStartPost,omitempty" systemd:"ExecStartPost,omitempty"`                                             // Commands or scripts that are executed after ExecStart. See related (ExecStart, ExecStartPre, ExecStartPost, ExecStop, ExecReload)
	ExecStop                 string `json:"ExecStop,omitempty" yaml:"ExecStop,omitempty" ini:"ExecStop,omitempty" systemd:"ExecStop,omitempty"`                                                                 // Command or script executed when the service is stopped. See related (ExecStart, ExecStartPre, ExecStartPost, ExecStop, ExecReload)
	ExecReload               string `json:"ExecReload,omitempty" yaml:"ExecReload,omitempty" ini:"ExecReload,omitempty" systemd:"ExecReload,omitempty"`                                                         // Command or script executed to reload the service's configuration without stopping it. See related (ExecStart, ExecStartPre, ExecStartPost, ExecStop, ExecReload)
	RemainAfterExit          string `json:"RemainAfterExit,omitempty" yaml:"RemainAfterExit,omitempty" ini:"RemainAfterExit,omitempty" systemd:"RemainAfterExit,omitempty"`                                     // The RemainAfterExit directive tells systemd how to treat the service once its main process exits. By default, systemd considers a service to be active if its main process is running. Once the main process exits, systemd usually marks the service as inactive. However, when RemainAfterExit is set to yes, systemd treats the service as still active even after its main process has exited.
	Restart                  string `json:"Restart,omitempty" yaml:"Restart,omitempty" ini:"Restart,omitempty" systemd:"Restart,omitempty"`                                                                     // Configures whether the service should be restarted when the service process exits, is killed, or a timeout is reached. Common values are `always`, `on-success`, `on-failure`, `on-abnormal`, `on-watchdog`, `on-abort`, and `never`. Defaults to "no".
	TimeoutSec               string `json:"TimeoutSec,omitempty" yaml:"TimeoutSec,omitempty" ini:"TimeoutSec,omitempty" systemd:"TimeoutSec,omitempty"`                                                         // Configure the time to wait for startup, shutdown, or overall operation respectively before marking the service as failed. See related (TimeoutSec, TimeoutStartSec, TimeoutStopSec)
	TimeoutStartSec          string `json:"TimeoutStartSec,omitempty" yaml:"TimeoutStartSec,omitempty" ini:"TimeoutStartSec,omitempty" systemd:"TimeoutStartSec,omitempty"`                                     // Configure the time to wait for startup. See related (TimeoutSec, TimeoutStartSec, TimeoutStopSec). Defaults to 90 seconds
	TimeoutStopSec           string `json:"TimeoutStopSec,omitempty" yaml:"TimeoutStopSec,omitempty" ini:"TimeoutStopSec,omitempty" systemd:"TimeoutSec,omitempty"`                                             // Configure the time to wait for stopping. See related (TimeoutSec, TimeoutStartSec, TimeoutStopSec). Defaults to 90 seconds
	Environment              string `json:"Environment,omitempty" yaml:"Environment,omitempty" ini:"Environment,omitempty" systemd:"Environment,omitempty"`                                                     // Sets environment variables for the service.
	EnvironmentFile          string `json:"EnvironmentFile,omitempty" yaml:"EnvironmentFile,omitempty" ini:"EnvironmentFile,omitempty" systemd:"EnvironmentFile,omitempty"`                                     // Sets environment variables from a file.
	WorkingDirectory         string `json:"WorkingDirectory,omitempty" yaml:"WorkingDirectory,omitempty" ini:"WorkingDirectory,omitempty" systemd:"WorkingDirectory,omitempty"`                                 // Sets the working directory for the service. Defaults to the root directory if not specified.
	RootDirectory            string `json:"RootDirectory,omitempty" yaml:"RootDirectory,omitempty" ini:"RootDirectory,omitempty" systemd:"RootDirectory,omitempty"`                                             // Sets the root directory for the service, changing the file system root for the executed processes.
	User                     string `json:"User,omitempty" yaml:"User,omitempty" ini:"User,omitempty" systemd:"User,omitempty"`                                                                                 // Sets the UNIX user that the service will run as. See related (User, Group)
	Group                    string `json:"Group,omitempty" yaml:"Group,omitempty" ini:"Group,omitempty" systemd:"Group,omitempty"`                                                                             // Sets the UNIX group that the service will run as. See related (User, Group)
	UMask                    string `json:"UMask,omitempty" yaml:"UMask,omitempty" ini:"UMask,omitempty" systemd:"UMask,omitempty"`                                                                             // Sets the UNIX file mode creation mask for the service. Defaults to 0022
	StandardError            string `json:"StandardError,omitempty" yaml:"StandardError,omitempty" ini:"StandardError,omitempty" systemd:"StandardError,omitempty"`                                             // Controls where file descriptor 2 (stderr) of the executed processes is connected to. The available options are identical to those of StandardOutput=, with some exceptions: if set to inherit the file descriptor used for standard output is duplicated for standard error, while fd:name will use a default file descriptor name of "stderr". See [official documentation](https://www.freedesktop.org/software/systemd/man/latest/systemd.exec.html#StandardError=)
	StandardInput            string `json:"StandardInput,omitempty" yaml:"StandardInput,omitempty" ini:"StandardInput,omitempty" systemd:"StandardInput,omitempty"`                                             // Controls where file descriptor 0 (STDIN) of the executed processes is connected to. Takes one of null, tty, tty-force, tty-fail, data, file:path, socket or fd:name. See [official documentation](https://www.freedesktop.org/software/systemd/man/latest/systemd.exec.html#StandardInput=).
	StandardOutput           string `json:"StandardOutput,omitempty" yaml:"StandardOutput,omitempty" ini:"StandardOutput,omitempty" systemd:"StandardOutput,omitempty"`                                         // Controls where file descriptor 1 (stdout) of the executed processes is connected to. Takes one of inherit, null, tty, journal, kmsg, journal+console, kmsg+console, file:path, append:path, truncate:path, socket or fd:name. See [official documentation](https://www.freedesktop.org/software/systemd/man/latest/systemd.exec.html#StandardOutput=)
	LimitNOFILE              string `json:"LimitNOFILE,omitempty" yaml:"LimitNOFILE,omitempty" ini:"LimitNOFILE,omitempty" systemd:"LimitNOFILE,omitempty"`                                                     // Set resource limits for the processes of this service, such as the number of open files or the number of processes. See related (LimitNOFILE, LimitNPROC) TODO - Refine Description
	LimitNPROC               string `json:"LimitNPROC,omitempty" yaml:"LimitNPROC,omitempty" ini:"LimitNPROC,omitempty" systemd:"LimitNPROC,omitempty"`                                                         // Set resource limits for the processes of this service, such as the number of open files or the number of processes. See related (LimitNOFILE, LimitNPROC) TODO - Refine Description
	RestartSec               string `json:"RestartSec,omitempty" yaml:"RestartSec,omitempty" ini:"RestartSec,omitempty" systemd:"RestartSec,omitempty"`                                                         // Sets the time to sleep before restarting a service (used with Restart). Defaults to 100 milliseconds
	SuccessExitStatus        string `json:"SuccessExitStatus,omitempty" yaml:"SuccessExitStatus,omitempty" ini:"SuccessExitStatus,omitempty" systemd:"SuccessExitStatus,omitempty"`                             // Sets the exit codes that will be considered as a successful service exit. See related (SuccessExitStatus, RestartPreventExitStatus, RestartForceExitStatus). Defaults to 0, SIGTERM, and SIGINT
	RestartPreventExitStatus string `json:"RestartPreventExitStatus,omitempty" yaml:"RestartPreventExitStatus,omitempty" ini:"RestartPreventExitStatus,omitempty" systemd:"RestartPreventExitStatus,omitempty"` // Sets the exit codes that will prevent automatic service restart when Restart is set to any of the automatic restart options. See related (SuccessExitStatus, RestartPreventExitStatus, RestartForceExitStatus)
	RestartForceExitStatus   string `json:"RestartForceExitStatus,omitempty" yaml:"RestartForceExitStatus,omitempty" ini:"RestartForceExitStatus,omitempty" systemd:"RestartForceExitStatus,omitempty"`         // Sets the exit codes that will force the service to restart even if `Restart` is set to `no`. See related (SuccessExitStatus, RestartPreventExitStatus, RestartForceExitStatus)
	PermissionsStartOnly     string `json:"PermissionsStartOnly,omitempty" yaml:"PermissionsStartOnly,omitempty" ini:"PermissionsStartOnly,omitempty" systemd:"PermissionsStartOnly,omitempty"`                 // If true, the root directory and user/group settings only apply to the ExecStart command, not to the various ExecStartPre, ExecStartPost, ExecReload, ExecStop, and ExecStopPost commands.
	RootDirectoryStartOnly   string `json:"RootDirectoryStartOnly,omitempty" yaml:"RootDirectoryStartOnly,omitempty" ini:"RootDirectoryStartOnly,omitempty" systemd:"RootDirectoryStartOnly,omitempty"`         // Similar to PermissionsStartOnly but applies to the RootDirectory setting.
	NonBlocking              string `json:"NonBlocking,omitempty" yaml:"NonBlocking,omitempty" ini:"NonBlocking,omitempty" systemd:"NonBlocking,omitempty"`                                                     // If true, all file descriptors except standard input, output, and error will be marked as non-blocking before executing the service's processes.
	NotifyAccess             string `json:"NotifyAccess,omitempty" yaml:"NotifyAccess,omitempty" ini:"NotifyAccess,omitempty" systemd:"NotifyAccess,omitempty"`                                                 // Configures how the service manager shall be notified about the service's start-up completion and runtime status. Common values are `none`, `main`, and `all`.
	Sockets                  string `json:"Sockets,omitempty" yaml:"Sockets,omitempty" ini:"Sockets,omitempty" systemd:"Sockets,omitempty"`                                                                     // Lists socket units that, when the service is started, will be passed to the service process.
	SuccessAction            string `json:"SuccessAction,omitempty" yaml:"SuccessAction,omitempty" ini:"SuccessAction,omitempty" systemd:"SuccessAction,omitempty"`                                             // Configure what action to take when the service fails or succeeds, respectively. See related (SuccessAction, FailureAction) TODO - Refine Description
	FailureAction            string `json:"FailureAction,omitempty" yaml:"FailureAction,omitempty" ini:"FailureAction,omitempty" systemd:"FailureAction,omitempty"`                                             // Configure what action to take when the service fails or succeeds, respectively. See related (SuccessAction, FailureAction) TODO - Refine Description
	CPUWeight                string `json:"CPUWeight,omitempty" yaml:"CPUWeight,omitempty" ini:"CPUWeight,omitempty" systemd:"CPUWeight,omitempty"`                                                             // resource control options: Set various resource control parameters for the service, influencing CPU, memory, and other resources allocation. See related (CPUWeight, StartupCPUWeight, CPUQuota, MemoryLimit, TasksMax) TODO - Refine Description
	StartupCPUWeight         string `json:"StartupCPUWeight,omitempty" yaml:"StartupCPUWeight,omitempty" ini:"StartupCPUWeight,omitempty" systemd:"StartupCPUWeight,omitempty"`                                 // resource control options: Set various resource control parameters for the service, influencing CPU, memory, and other resources allocation. See related (CPUWeight, StartupCPUWeight, CPUQuota, MemoryLimit, TasksMax) TODO - Refine Description
	CPUQuota                 string `json:"CPUQuota,omitempty" yaml:"CPUQuota,omitempty" ini:"CPUQuota,omitempty" systemd:"CPUQuota,omitempty"`                                                                 // resource control options: Set various resource control parameters for the service, influencing CPU, memory, and other resources allocation. See related (CPUWeight, StartupCPUWeight, CPUQuota, MemoryLimit, TasksMax) TODO - Refine Description
	MemoryLimit              string `json:"MemoryLimit,omitempty" yaml:"MemoryLimit,omitempty" ini:"MemoryLimit,omitempty" systemd:"MemoryLimit,omitempty"`                                                     // resource control options: Set various resource control parameters for the service, influencing CPU, memory, and other resources allocation. See related (CPUWeight, StartupCPUWeight, CPUQuota, MemoryLimit, TasksMax) TODO - Refine Description
	TasksMax                 string `json:"TasksMax,omitempty" yaml:"TasksMax,omitempty" ini:"TasksMax,omitempty" systemd:"TasksMax,omitempty"`                                                                 // resource control options: Set various resource control parameters for the service, influencing CPU, memory, and other resources allocation. See related (CPUWeight, StartupCPUWeight, CPUQuota, MemoryLimit, TasksMax) TODO - Refine Description
	AmbientCapabilities      string `json:"AmbientCapabilities,omitempty" yaml:"AmbientCapabilities,omitempty" ini:"AmbientCapabilities,omitempty" systemd:"AmbientCapabilities,omitempty"`                     // Sets additional capabilities for the service process.
	CapabilityBoundingSet    string `json:"CapabilityBoundingSet,omitempty" yaml:"CapabilityBoundingSet,omitempty" ini:"CapabilityBoundingSet,omitempty" systemd:"CapabilityBoundingSet,omitempty"`             // Controls which capabilities the service process retains.
	ProtectSystem            string `json:"ProtectSystem,omitempty" yaml:"ProtectSystem,omitempty" ini:"ProtectSystem,omitempty" systemd:"ProtectSystem,omitempty"`                                             // security-related options: Provide different levels of security and isolation for the service by restricting access to various system features and components. See related (ProtectSystem, ProtectHome, PrivateTmp, PrivateDevices, PrivateNetwork) TODO - Refine Description
	ProtectHome              string `json:"ProtectHome,omitempty" yaml:"ProtectHome,omitempty" ini:"ProtectHome,omitempty" systemd:"ProtectHome,omitempty"`                                                     // security-related options: Provide different levels of security and isolation for the service by restricting access to various system features and components. See related (ProtectSystem, ProtectHome, PrivateTmp, PrivateDevices, PrivateNetwork) TODO - Refine Description
	PrivateTmp               string `json:"PrivateTmp,omitempty" yaml:"PrivateTmp,omitempty" ini:"PrivateTmp,omitempty" systemd:"PrivateTmp,omitempty"`                                                         // security-related options: Provide different levels of security and isolation for the service by restricting access to various system features and components. See related (ProtectSystem, ProtectHome, PrivateTmp, PrivateDevices, PrivateNetwork) TODO - Refine Description
	PrivateDevices           string `json:"PrivateDevices,omitempty" yaml:"PrivateDevices,omitempty" ini:"PrivateDevices,omitempty" systemd:"PrivateDevices,omitempty"`                                         // security-related options: Provide different levels of security and isolation for the service by restricting access to various system features and components. See related (ProtectSystem, ProtectHome, PrivateTmp, PrivateDevices, PrivateNetwork) TODO - Refine Description
	PrivateNetwork           string `json:"PrivateNetwork,omitempty" yaml:"PrivateNetwork,omitempty" ini:"PrivateNetwork,omitempty" systemd:"PrivateNetwork,omitempty"`                                         // security-related options: Provide different levels of security and isolation for the service by restricting access to various system features and components. See related (ProtectSystem, ProtectHome, PrivateTmp, PrivateDevices, PrivateNetwork) TODO - Refine Description
	ReadWritePaths           string `json:"ReadWritePaths,omitempty" yaml:"ReadWritePaths,omitempty" ini:"ReadWritePaths,omitempty" systemd:"ReadWritePaths,omitempty"`                                         // Configure specific directories to be read-write, read-only, or inaccessible to the service. TODO - Refine Description
	ReadOnlyPaths            string `json:"ReadOnlyPaths,omitempty" yaml:"ReadOnlyPaths,omitempty" ini:"ReadOnlyPaths,omitempty" systemd:"ReadOnlyPaths,omitempty"`                                             // Configure specific directories to be read-write, read-only, or inaccessible to the service. TODO - Refine Description
	InaccessiblePaths        string `json:"InaccessiblePaths,omitempty" yaml:"InaccessiblePaths,omitempty" ini:"InaccessiblePaths,omitempty" systemd:"InaccessiblePaths,omitempty"`                             // Configure specific directories to be read-write, read-only, or inaccessible to the service. TODO - Refine Description
	NoNewPrivileges          string `json:"NoNewPrivileges,omitempty" yaml:"NoNewPrivileges,omitempty" ini:"NoNewPrivileges,omitempty" systemd:"NoNewPrivileges,omitempty"`                                     // If true, ensures that the service processes cannot gain new privileges.
}

func (s *Service) tags() (tags []*tag) {
	const key = "systemd"

	tags = make([]*tag, 0)
	instance := reflect.ValueOf(s).Elem()
	structure := reflect.TypeOf(s).Elem()

	for i := 0; i < structure.NumField(); i++ {
		field := structure.Field(i)
		if v, ok := field.Tag.Lookup(key); ok {
			partials := strings.Split(v, ",")
			for idx, partial := range partials {
				partials[idx] = strings.TrimSpace(partial)
			}

			var optional bool
			if len(partials) > 1 {
				for _, partial := range partials {
					partial = strings.ToLower(partial)
					if partial == "omitempty" {
						optional = true
					}
				}
			}

			var assignment *string
			if value := instance.Field(i).String(); value != "" {
				assignment = &value
			}

			attribute := &tag{
				Name:     partials[0],
				Field:    field.Name,
				Optional: optional,
				Value:    assignment,
			}

			tags = append(tags, attribute)
		}
	}

	return
}

// assignments represents the set of key-values to write to a systemd file.
func (s *Service) assignments() (exports map[string]string) {
	exports = make(map[string]string)

	for _, tag := range s.tags() {
		if !(tag.Optional) || tag.Value != nil {
			exports[tag.Name] = *tag.Value
		}
	}

	return
}

// export represents the service's raw buffer of a [Service] section.
func (s *Service) export() (*bytes.Buffer, error) {
	file := ini.Empty()
	section, e := file.NewSection("Service")
	if e != nil {
		return nil, e
	}

	for key, value := range s.assignments() {
		if _, e := section.NewKey(key, value); e != nil {
			return nil, e
		}
	}

	var buffer, output bytes.Buffer
	if _, e := file.WriteTo(&buffer); e != nil {
		return nil, e
	}

	scanner := bufio.NewScanner(&buffer)
	for scanner.Scan() {
		line := scanner.Bytes()
		if key, value, valid := bytes.Cut(line, []byte("=")); valid {
			var array = [][]byte{bytes.TrimSpace(key), bytes.TrimSpace(value)}

			line = append(bytes.Join(array, []byte("=")), []byte("\n")...)
			if _, e := output.Write(line); e != nil {
				return nil, e
			}

			continue
		}

		output.Write(append(line, []byte("\n")...))
	}

	if e := scanner.Err(); e != nil {
		return nil, e
	}

	if _, e := output.Write([]byte("\n")); e != nil {
		return nil, e
	}

	return &output, nil
}

// Install represents the [Install] section of a systemd service file.
//
// The [Install] section of a systemd service file is used to define how the service should be installed and integrated into the system's boot sequence.
// This section is primarily used by the `systemctl enable` and `systemctl disable` commands to manage the links for service files. Here's a list of options
// you might find in the `[Install]` section, along with their descriptions:
//
// These directives control how and when a service starts during the system's boot process or when certain conditions are met. The `[Install]` section does not
// influence the service's runtime behavior but instead controls how the service is set up within the system's service management framework.
//
//   - Directives such as WantedBy and RequiredBy can contain multiple definitions on a single line. E.g. WantedBy=multi-user.target docker.service
type Install struct {
	WantedBy        string `json:"WantedBy,omitempty" yaml:"WantedBy,omitempty" ini:"WantedBy,omitempty" systemd:"WantedBy,omitempty"`                             // Specifies the target or targets that the unit should be added to as a dependency when enabled. This is probably the most commonly used directive in the `[Install]` section. For instance, setting `WantedBy=multi-user.target` means the service will start at the multi-user runlevel.
	RequiredBy      string `json:"RequiredBy,omitempty" yaml:"RequiredBy,omitempty" ini:"RequiredBy,omitempty" systemd:"RequiredBy,omitempty"`                     // Similar to `WantedBy`, but creates a stronger dependency. Units listed here will fail to start if the service fails to start.
	Alias           string `json:"Alias,omitempty" yaml:"Alias,omitempty" ini:"Alias,omitempty" systemd:"Alias,omitempty"`                                         // Provides a space-separated list of additional names for the unit. When the unit is enabled, symlinks will be created for these names as well.
	Also            string `json:"Also,omitempty" yaml:"Also,omitempty" ini:"Also,omitempty" systemd:"Also,omitempty"`                                             // Specifies additional units that should be enabled or disabled whenever this unit is enabled or disabled.
	DefaultInstance string `json:"DefaultInstance,omitempty" yaml:"DefaultInstance,omitempty" ini:"DefaultInstance,omitempty" systemd:"DefaultInstance,omitempty"` // For template units, this sets the default instance name used when no instance name is specified.
}

func (i *Install) tags() (tags []*tag) {
	const key = "systemd"

	tags = make([]*tag, 0)
	instance := reflect.ValueOf(i).Elem()
	structure := reflect.TypeOf(i).Elem()

	for idx := 0; idx < structure.NumField(); idx++ {
		field := structure.Field(idx)
		if v, ok := field.Tag.Lookup(key); ok {
			partials := strings.Split(v, ",")
			for idx, partial := range partials {
				partials[idx] = strings.TrimSpace(partial)
			}

			var optional bool
			if len(partials) > 1 {
				for _, partial := range partials {
					partial = strings.ToLower(partial)
					if partial == "omitempty" {
						optional = true
					}
				}
			}

			var assignment *string
			if value := instance.Field(idx).String(); value != "" {
				assignment = &value
			}

			attribute := &tag{
				Name:     partials[0],
				Field:    field.Name,
				Optional: optional,
				Value:    assignment,
			}

			tags = append(tags, attribute)
		}
	}

	return
}

// assignments represents the set of key-values to write to a systemd file.
func (i *Install) assignments() (exports map[string]string) {
	exports = make(map[string]string)

	for _, tag := range i.tags() {
		if !(tag.Optional) || tag.Value != nil {
			exports[tag.Name] = *tag.Value
		}
	}

	return
}

// export represents the service's raw buffer of a [Service] section.
func (i *Install) export() (*bytes.Buffer, error) {
	file := ini.Empty()
	section, e := file.NewSection("Install")
	if e != nil {
		return nil, e
	}

	for key, value := range i.assignments() {
		if _, e := section.NewKey(key, value); e != nil {
			return nil, e
		}
	}

	var buffer, output bytes.Buffer
	if _, e := file.WriteTo(&buffer); e != nil {
		return nil, e
	}

	scanner := bufio.NewScanner(&buffer)
	for scanner.Scan() {
		line := scanner.Bytes()
		if key, value, valid := bytes.Cut(line, []byte("=")); valid {
			var array = [][]byte{bytes.TrimSpace(key), bytes.TrimSpace(value)}

			line = append(bytes.Join(array, []byte("=")), []byte("\n")...)
			if _, e := output.Write(line); e != nil {
				return nil, e
			}

			continue
		}

		output.Write(append(line, []byte("\n")...))
	}

	if e := scanner.Err(); e != nil {
		return nil, e
	}

	if _, e := output.Write([]byte("\n")); e != nil {
		return nil, e
	}

	return &output, nil
}

// Socket - The [Socket] section of a systemd service file is used to define socket-based activation for a service. This feature of systemd allows a service to be
// started on-demand when a particular socket or network connection is accessed. Here are the common options available in the `[Socket]` section, along with
// their descriptions:
//
// These options allow you to configure how the socket associated with your service behaves, including who can access it, how data is handled, and when the
// service associated with the socket is started. Socket-based activation can significantly improve the efficiency and responsiveness of your services,
// particularly for services that need to handle a large number of incoming connections.
type Socket struct {
	ListenStream            string `json:"ListenStream,omitempty" yaml:"ListenStream,omitempty" ini:"ListenStream,omitempty" systemd:"ListenStream,omitempty"`                                             // Specifies a file system socket, or a network socket, for stream (i.e., TCP) sockets. The service is started when a connection is made to this socket.
	ListenDatagram          string `json:"ListenDatagram,omitempty" yaml:"ListenDatagram,omitempty" ini:"ListenDatagram,omitempty" systemd:"ListenDatagram,omitempty"`                                     // Similar to ListenStream, but for datagram (i.e., UDP) sockets.
	ListenSequentialPacket  string `json:"ListenSequentialPacket,omitempty" yaml:"ListenSequentialPacket,omitempty" ini:"ListenSequentialPacket,omitempty" systemd:"ListenSequentialPacket,omitempty"`     // Specifies a path for a sequential packet socket, or a network socket, for stream-oriented packets.
	ListenFIFO              string `json:"ListenFIFO,omitempty" yaml:"ListenFIFO,omitempty" ini:"ListenFIFO,omitempty" systemd:"ListenFIFO,omitempty"`                                                     // Specifies a FIFO (named pipe) for the socket. The service is started when data is written to the FIFO.
	ListenSpecial           string `json:"ListenSpecial,omitempty" yaml:"ListenSpecial,omitempty" ini:"ListenSpecial,omitempty" systemd:"ListenSpecial,omitempty"`                                         // Specifies a special file, such as a device node, for the socket. The service is started when the special file is accessed.
	ListenNetlink           string `json:"ListenNetlink,omitempty" yaml:"ListenNetlink,omitempty" ini:"ListenNetlink,omitempty" systemd:"ListenNetlink,omitempty"`                                         // Specifies a Netlink socket. This is used for IPC (Inter-Process Communication) between kernel and userspace processes.
	ListenMessageQueue      string `json:"ListenMessageQueue,omitempty" yaml:"ListenMessageQueue,omitempty" ini:"ListenMessageQueue,omitempty" systemd:"ListenMessageQueue,omitempty"`                     // Specifies a POSIX message queue. The service is triggered when a message is sent to the queue.
	SocketMode              string `json:"SocketMode,omitempty" yaml:"SocketMode,omitempty" ini:"SocketMode,omitempty" systemd:"SocketMode,omitempty"`                                                     // Sets the file system access mode used for the socket file.
	SocketUser              string `json:"SocketUser,omitempty" yaml:"SocketUser,omitempty" ini:"SocketUser,omitempty" systemd:"SocketUser,omitempty"`                                                     // Specify the UNIX user that own the socket file.
	SocketGroup             string `json:"SocketGroup,omitempty" yaml:"SocketGroup,omitempty" ini:"SocketGroup,omitempty" systemd:"SocketGroup,omitempty"`                                                 // // Specify the UNIX group that own the socket file.
	SocketProtocol          string `json:"SocketProtocol,omitempty" yaml:"SocketProtocol,omitempty" ini:"SocketProtocol,omitempty" systemd:"SocketProtocol,omitempty"`                                     // Sets the protocol used for the socket, applicable for certain types of sockets like Netlink.
	BindToDevice            string `json:"BindToDevice,omitempty" yaml:"BindToDevice,omitempty" ini:"BindToDevice,omitempty" systemd:"BindToDevice,omitempty"`                                             // Binds the socket to a specific network device.
	Service                 string `json:"Service,omitempty" yaml:"Service,omitempty" ini:"Service,omitempty" systemd:"Service,omitempty"`                                                                 // Specifies the service unit that is started when the socket receives activity.
	PassCredentials         string `json:"PassCredentials,omitempty" yaml:"PassCredentials,omitempty" ini:"PassCredentials,omitempty" systemd:"PassCredentials,omitempty"`                                 // A boolean that specifies whether the socket should pass credentials (such as PID, UID, and GID) when a service is spawned.
	PassSecurity            string `json:"PassSecurity,omitempty" yaml:"PassSecurity,omitempty" ini:"PassSecurity,omitempty" systemd:"PassSecurity,omitempty"`                                             // A boolean that specifies whether the socket should pass security-related information when a service is spawned.
	ReceiveBuffer           string `json:"ReceiveBuffer,omitempty" yaml:"ReceiveBuffer,omitempty" ini:"ReceiveBuffer,omitempty" systemd:"ReceiveBuffer,omitempty"`                                         // Set the size of the receive buffer for the socket.
	SendBuffer              string `json:"SendBuffer,omitempty" yaml:"SendBuffer,omitempty" ini:"SendBuffer,omitempty" systemd:"SendBuffer,omitempty"`                                                     // Set the size of the send buffer for the socket.
	MaxConnections          string `json:"MaxConnections,omitempty" yaml:"MaxConnections,omitempty" ini:"MaxConnections,omitempty" systemd:"MaxConnections,omitempty"`                                     // Sets the maximum number of connections that will be queued for the socket.
	MaxConnectionsPerSource string `json:"MaxConnectionsPerSource,omitempty" yaml:"MaxConnectionsPerSource,omitempty" ini:"MaxConnectionsPerSource,omitempty" systemd:"MaxConnectionsPerSource,omitempty"` // Sets the maximum number of connections per source IP for this socket.
	KeepAlive               string `json:"KeepAlive,omitempty" yaml:"KeepAlive,omitempty" ini:"KeepAlive,omitempty" systemd:"KeepAlive,omitempty"`                                                         // Configure TCP keepalive parameters for the socket. See related (KeepAlive, KeepAliveTimeSec, KeepAliveIntervalSec, KeepAliveProbes) TODO - Refine descriptions
	KeepAliveTimeSec        string `json:"KeepAliveTimeSec,omitempty" yaml:"KeepAliveTimeSec,omitempty" ini:"KeepAliveTimeSec,omitempty" systemd:"KeepAliveTimeSec,omitempty"`                             // Configure TCP keepalive parameters for the socket. See related (KeepAlive, KeepAliveTimeSec, KeepAliveIntervalSec, KeepAliveProbes) TODO - Refine descriptions
	KeepAliveIntervalSec    string `json:"KeepAliveIntervalSec,omitempty" yaml:"KeepAliveIntervalSec,omitempty" ini:"KeepAliveIntervalSec,omitempty" systemd:"KeepAliveIntervalSec,omitempty"`             // Configure TCP keepalive parameters for the socket. See related (KeepAlive, KeepAliveTimeSec, KeepAliveIntervalSec, KeepAliveProbes) TODO - Refine descriptions
	KeepAliveProbes         string `json:"KeepAliveProbes,omitempty" yaml:"KeepAliveProbes,omitempty" ini:"KeepAliveProbes,omitempty" systemd:"KeepAliveProbes,omitempty"`                                 // Configure TCP keepalive parameters for the socket. See related (KeepAlive, KeepAliveTimeSec, KeepAliveIntervalSec, KeepAliveProbes) TODO - Refine descriptions
	NoDelay                 string `json:"NoDelay,omitempty" yaml:"NoDelay,omitempty" ini:"NoDelay,omitempty" systemd:"NoDelay,omitempty"`                                                                 // A boolean option that controls the TCP_NODELAY socket option, which disables the Nagle algorithm for send coalescing.
	Priority                string `json:"Priority,omitempty" yaml:"Priority,omitempty" ini:"Priority,omitempty" systemd:"Priority,omitempty"`                                                             // Sets the priority of the socket, which can affect the scheduling of packets for network sockets.
	DeferAcceptSec          string `json:"DeferAcceptSec,omitempty" yaml:"DeferAcceptSec,omitempty" ini:"DeferAcceptSec,omitempty" systemd:"DeferAcceptSec,omitempty"`                                     // Delays the connection from being accepted until data is available, reducing resource usage for services.
	Accept                  string `json:"Accept,omitempty" yaml:"Accept,omitempty" ini:"Accept,omitempty" systemd:"Accept,omitempty"`                                                                     // A boolean that specifies whether an individual service instance is spawned for each incoming connection (when true) or if connections should be accepted by the main service (when false).
	Writable                string `json:"Writable,omitempty" yaml:"Writable,omitempty" ini:"Writable,omitempty" systemd:"Writable,omitempty"`                                                             // A boolean that specifies whether the socket file should be writable.
	TriggerLimitIntervalSec string `json:"TriggerLimitIntervalSec,omitempty" yaml:"TriggerLimitIntervalSec,omitempty" ini:"TriggerLimitIntervalSec,omitempty" systemd:"TriggerLimitIntervalSec,omitempty"` // Configure rate limiting for activation requests. See related TriggerLimitBurst
	TriggerLimitBurst       string `json:"TriggerLimitBurst,omitempty" yaml:"TriggerLimitBurst,omitempty" ini:"TriggerLimitBurst,omitempty" systemd:"TriggerLimitBurst,omitempty"`                         // Configure rate limiting for activation requests. See related TriggerLimitIntervalSec
}

func (s *Socket) tags() (tags []*tag) {
	const key = "systemd"

	tags = make([]*tag, 0)
	instance := reflect.ValueOf(s).Elem()
	structure := reflect.TypeOf(s).Elem()

	for i := 0; i < structure.NumField(); i++ {
		field := structure.Field(i)
		if v, ok := field.Tag.Lookup(key); ok {
			partials := strings.Split(v, ",")
			for idx, partial := range partials {
				partials[idx] = strings.TrimSpace(partial)
			}

			var optional bool
			if len(partials) > 1 {
				for _, partial := range partials {
					partial = strings.ToLower(partial)
					if partial == "omitempty" {
						optional = true
					}
				}
			}

			var assignment *string
			if value := instance.Field(i).String(); value != "" {
				assignment = &value
			}

			attribute := &tag{
				Name:     partials[0],
				Field:    field.Name,
				Optional: optional,
				Value:    assignment,
			}

			tags = append(tags, attribute)
		}
	}

	return
}

// assignments represents the set of key-values to write to a systemd file.
func (s *Socket) assignments() (exports map[string]string) {
	exports = make(map[string]string)

	for _, tag := range s.tags() {
		if !(tag.Optional) || tag.Value != nil {
			exports[tag.Name] = *tag.Value
		}
	}

	return
}

// export represents the service's raw buffer of a [Service] section.
func (s *Socket) export() (*bytes.Buffer, error) {
	file := ini.Empty()
	section, e := file.NewSection("Socket")
	if e != nil {
		return nil, e
	}

	for key, value := range s.assignments() {
		if _, e := section.NewKey(key, value); e != nil {
			return nil, e
		}
	}

	var buffer, output bytes.Buffer
	if _, e := file.WriteTo(&buffer); e != nil {
		return nil, e
	}

	scanner := bufio.NewScanner(&buffer)
	for scanner.Scan() {
		line := scanner.Bytes()
		if key, value, valid := bytes.Cut(line, []byte("=")); valid {
			var array = [][]byte{bytes.TrimSpace(key), bytes.TrimSpace(value)}

			line = append(bytes.Join(array, []byte("=")), []byte("\n")...)
			if _, e := output.Write(line); e != nil {
				return nil, e
			}

			continue
		}

		output.Write(append(line, []byte("\n")...))
	}

	if e := scanner.Err(); e != nil {
		return nil, e
	}

	if _, e := output.Write([]byte("\n")); e != nil {
		return nil, e
	}

	return &output, nil
}

// Daemon represents a complete systemd service file configuration.
//
//   - Note that "booleans" in systemd can be either "yes", "no", "true" or "false
//
// See [systemd] for additional, high-level information, [directives] for an exhaustive list of options, [defaults] for
// a larger list of default settings.
//
// [systemd]: https://www.freedesktop.org/software/systemd/man/latest/systemd.service.html
// [defaults]: https://www.freedesktop.org/software/systemd/man/latest/systemd-system.conf.html#
// [directives]: https://www.freedesktop.org/software/systemd/man/latest/systemd.directives.html
type Daemon struct {
	Unit    Unit    `json:"Unit" yaml:"Unit" ini:"Unit" systemd:"Unit"`
	Service Service `json:"Service" yaml:"Service" ini:"Service" systemd:"Service"`
	Install Install `json:"Install" yaml:"Install" ini:"Install" systemd:"Install"`
	Socket  *Socket `json:"Socket,omitempty" yaml:"Socket,omitempty" ini:"Socket,omitempty" systemd:"Socket,omitempty"`
}

func (d *Daemon) MarshalText() ([]byte, error) {
	var exceptions = make([]error, 0)

	unit, e := d.Unit.export()
	if e != nil {
		exceptions = append(exceptions, e)
	}

	service, e := d.Service.export()
	if e != nil {
		exceptions = append(exceptions, e)
	}

	install, e := d.Install.export()
	if e != nil {
		exceptions = append(exceptions, e)
	}

	if len(exceptions) > 0 {
		return nil, errors.Join(exceptions...)
	}

	readers := []io.Reader{unit, service, install}
	if socket := d.Socket; socket != nil {
		v, e := socket.export()
		if e != nil {
			return nil, e
		}

		readers = append(readers, v)
	}

	reader := io.MultiReader(readers...)

	content, e := io.ReadAll(reader)
	if e != nil {
		return nil, e
	}

	return bytes.TrimSpace(content), nil
}

func (d *Daemon) UnmarshalText(stream []byte) error {
	file, e := ini.Load(stream)
	if e != nil {
		return fmt.Errorf("unable to unmarshal daemon file: %w", e)
	}

	var unit Unit
	if section := file.Section("Unit"); section == nil {
		return fmt.Errorf("invalid [Unit] systemd section")
	} else if e := section.MapTo(&unit); e != nil {
		return fmt.Errorf("unable to unmarshal [Unit] systemd section: %w", e)
	}

	var service Service
	if section := file.Section("Service"); section == nil {
		return fmt.Errorf("invalid [Service] systemd section")
	} else if e := section.MapTo(&service); e != nil {
		return fmt.Errorf("unable to unmarshal [Service] systemd section: %w", e)
	}

	var install Install
	if section := file.Section("Install"); section == nil {
		return fmt.Errorf("invalid [Install] systemd section")
	} else if e := section.MapTo(&install); e != nil {
		return fmt.Errorf("unable to unmarshal [Install] systemd section: %w", e)
	}

	var socket *Socket
	if section := file.Section("Socket"); file.HasSection("Socket") && section != nil {
		if len(section.KeysHash()) > 0 {
			if e := section.MapTo(socket); e != nil {
				return fmt.Errorf("unable to unmarshal [Socket] systemd section: %w", e)
			}
		}
	}

	instance := Daemon{unit, service, install, socket}

	*d = instance

	return nil
}

func Unmarshal(stream []byte) (*Daemon, error) {
	file, e := ini.Load(stream)
	if e != nil {
		return nil, fmt.Errorf("unable to unmarshal daemon file: %w", e)
	}

	var unit Unit
	if section := file.Section("Unit"); section == nil {
		return nil, fmt.Errorf("invalid [Unit] systemd section")
	} else if e := section.MapTo(&unit); e != nil {
		return nil, fmt.Errorf("unable to unmarshal [Unit] systemd section: %w", e)
	}

	var service Service
	if section := file.Section("Service"); section == nil {
		return nil, fmt.Errorf("invalid [Service] systemd section")
	} else if e := section.MapTo(&service); e != nil {
		return nil, fmt.Errorf("unable to unmarshal [Service] systemd section: %w", e)
	}

	var install Install
	if section := file.Section("Install"); section == nil {
		return nil, fmt.Errorf("invalid [Install] systemd section")
	} else if e := section.MapTo(&install); e != nil {
		return nil, fmt.Errorf("unable to unmarshal [Install] systemd section: %w", e)
	}

	var socket *Socket
	if section := file.Section("Socket"); file.HasSection("Socket") && section != nil {
		if len(section.KeysHash()) > 0 {
			if e := section.MapTo(socket); e != nil {
				return nil, fmt.Errorf("unable to unmarshal [Socket] systemd section: %w", e)
			}
		}
	}

	return &Daemon{unit, service, install, socket}, nil
}

func Marshal(systemd Daemon) ([]byte, error) {
	ini.PrettyEqual = false
	ini.PrettyFormat = false
	ini.PrettySection = true

	return systemd.MarshalText()
}
