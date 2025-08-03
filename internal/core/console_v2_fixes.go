package core

import (
	"fmt"
)

// rebuildCommandTreeForClarity rebuilds command tree with clear separation
func (c *ConsoleV2) rebuildCommandTreeForClarity() {
	// Root node
	c.rootCommand = NewCommandNode("strigoi", "Strigoi security validation platform")
	
	// Global commands (always available)
	c.addGlobalCommands()
	
	// Build main command categories
	c.buildStreamCommandsV2()
	c.buildIntegrationCommandsV2()
	c.buildProbeCommandsV2()
	c.buildSenseCommandsV2()
	c.buildRespondCommandsV2()
	c.buildReportCommandsV2()
	c.buildSupportCommandsV2()
	c.buildStateCommandsV2()
	c.buildJobsCommandV2()
}

// addGlobalCommands adds commands available everywhere
func (c *ConsoleV2) addGlobalCommands() {
	// Help command (visible)
	help := NewCommandNode("help", "Show help information")
	help.Handler = c.handleHelp
	help.AddArg(CommandArg{
		Name:        "command",
		Description: "Command path to get help for",
		Required:    false,
		Multiple:    true,
	})
	c.rootCommand.AddChild(help)
	
	// Exit command (visible)
	exit := NewCommandNode("exit", "Exit the console")
	exit.Handler = c.handleExit
	c.rootCommand.AddChild(exit)
	
	// Clear command (visible)
	clear := NewCommandNode("clear", "Clear the screen")
	clear.Handler = c.handleClear
	c.rootCommand.AddChild(clear)
	
	// Alias command (visible)
	alias := NewCommandNode("alias", "Manage command aliases")
	alias.Handler = c.handleAlias
	alias.AddArg(CommandArg{
		Name:        "alias",
		Description: "Alias name",
		Required:    false,
	})
	alias.AddArg(CommandArg{
		Name:        "command",
		Description: "Command to alias",
		Required:    false,
	})
	c.rootCommand.AddChild(alias)
	
	// Navigation commands (hidden but available everywhere)
	cd := NewCommandNode("cd", "Change directory")
	cd.Hidden = true // Hide from normal listing
	c.rootCommand.AddChild(cd)
	
	pwd := NewCommandNode("pwd", "Print working directory")
	pwd.Hidden = true
	c.rootCommand.AddChild(pwd)
	
	ls := NewCommandNode("ls", "List available commands and directories")
	ls.Hidden = true
	c.rootCommand.AddChild(ls)
}

// buildStreamCommandsV2 builds stream commands with clear structure
func (c *ConsoleV2) buildStreamCommandsV2() {
	// stream is a DIRECTORY containing commands
	stream := NewCommandNode("stream", "STDIO stream monitoring & analysis")
	
	// These are EXECUTABLE COMMANDS within stream/
	tap := NewCommandNode("tap", "Monitor process STDIO in real-time")
	tap.Handler = c.handleStreamTap
	tap.AddFlag(CommandFlag{
		Name:        "auto-discover",
		Short:       "a",
		Description: "Automatically discover Claude/MCP processes",
		Type:        "bool",
		Default:     "false",
	})
	tap.AddFlag(CommandFlag{
		Name:        "pid",
		Short:       "p",
		Description: "Process ID to monitor",
		Type:        "int",
	})
	tap.AddFlag(CommandFlag{
		Name:        "duration",
		Short:       "d",
		Description: "Monitoring duration (e.g., 30s, 5m)",
		Type:        "duration",
		Default:     "30s",
	})
	tap.AddFlag(CommandFlag{
		Name:        "output",
		Short:       "o",
		Description: "Output destination",
		Type:        "string",
		Default:     "stdout",
	})
	tap.AddExample("tap --auto-discover")
	tap.AddExample("tap --pid 12345 --duration 1m")
	stream.AddChild(tap)
	
	record := NewCommandNode("record", "Record streams for later analysis")
	record.Handler = c.handleStreamRecord
	stream.AddChild(record)
	
	status := NewCommandNode("status", "Show stream monitoring status")
	status.Handler = c.handleStreamStatus
	stream.AddChild(status)
	
	c.rootCommand.AddChild(stream)
}

// buildProbeCommandsV2 builds probe commands as a directory
func (c *ConsoleV2) buildProbeCommandsV2() {
	// probe is a DIRECTORY
	probe := NewCommandNode("probe", "Discovery and reconnaissance tools")
	
	// These are EXECUTABLE COMMANDS, not subdirectories
	north := NewCommandNode("north", "Probe north direction (endpoints)")
	north.Handler = c.handleProbeDirection
	north.AddFlag(CommandFlag{
		Name:        "depth",
		Short:       "d",
		Description: "Probe depth",
		Type:        "int",
		Default:     "1",
	})
	probe.AddChild(north)
	
	south := NewCommandNode("south", "Probe south direction (dependencies)")
	south.Handler = c.handleProbeDirection
	probe.AddChild(south)
	
	east := NewCommandNode("east", "Probe east direction (data flows)")
	east.Handler = c.handleProbeDirection
	probe.AddChild(east)
	
	west := NewCommandNode("west", "Probe west direction (integrations)")
	west.Handler = c.handleProbeDirection
	probe.AddChild(west)
	
	all := NewCommandNode("all", "Probe all directions")
	all.Handler = c.handleProbeAll
	probe.AddChild(all)
	
	c.rootCommand.AddChild(probe)
}

// buildIntegrationCommandsV2 with proper hierarchy
func (c *ConsoleV2) buildIntegrationCommandsV2() {
	// integrations is a DIRECTORY
	integrations := NewCommandNode("integrations", "External system integrations")
	
	// list is an EXECUTABLE COMMAND
	list := NewCommandNode("list", "List available integrations")
	list.Handler = c.handleIntegrationsList
	integrations.AddChild(list)
	
	// prometheus is a SUBDIRECTORY
	prometheus := NewCommandNode("prometheus", "Prometheus metrics integration")
	
	// Commands within prometheus/
	promEnable := NewCommandNode("enable", "Enable Prometheus metrics export")
	promEnable.Handler = c.handlePrometheusEnable
	promEnable.AddFlag(CommandFlag{
		Name:        "port",
		Short:       "p",
		Description: "HTTP port for metrics endpoint",
		Type:        "int",
		Default:     "9090",
	})
	prometheus.AddChild(promEnable)
	
	promDisable := NewCommandNode("disable", "Disable Prometheus metrics")
	promDisable.Handler = c.handlePrometheusDisable
	prometheus.AddChild(promDisable)
	
	promStatus := NewCommandNode("status", "Show Prometheus integration status")
	promStatus.Handler = c.handlePrometheusStatus
	prometheus.AddChild(promStatus)
	
	integrations.AddChild(prometheus)
	
	// syslog subdirectory
	syslog := NewCommandNode("syslog", "Syslog integration")
	syslogEnable := NewCommandNode("enable", "Enable syslog forwarding")
	syslogEnable.Handler = c.handleSyslogEnable
	syslog.AddChild(syslogEnable)
	integrations.AddChild(syslog)
	
	c.rootCommand.AddChild(integrations)
}

// buildSenseCommandsV2 builds sense commands
func (c *ConsoleV2) buildSenseCommandsV2() {
	// sense is a DIRECTORY
	sense := NewCommandNode("sense", "Analysis and interpretation tools")
	
	// These are EXECUTABLE COMMANDS
	for _, layer := range []string{"network", "transport", "protocol", "application"} {
		node := NewCommandNode(layer, fmt.Sprintf("Analyze %s layer", layer))
		node.Handler = c.handleSenseLayer
		sense.AddChild(node)
	}
	
	c.rootCommand.AddChild(sense)
}

// Stub handlers for new commands
func (c *ConsoleV2) handlePrometheusStatus(console interface{}, cmd *ParsedCommand) error {
	c.Info("Prometheus integration: disabled")
	return nil
}

func (c *ConsoleV2) handleSyslogEnable(console interface{}, cmd *ParsedCommand) error {
	c.Warn("Syslog integration not yet implemented")
	return nil
}

// buildRespondCommandsV2 builds respond commands
func (c *ConsoleV2) buildRespondCommandsV2() {
	// respond is a DIRECTORY (future implementation)
	respond := NewCommandNode("respond", "Response and mitigation tools")
	respond.Handler = c.handleRespondPlaceholder
	c.rootCommand.AddChild(respond)
}

// buildReportCommandsV2 builds report commands
func (c *ConsoleV2) buildReportCommandsV2() {
	// report is a DIRECTORY
	report := NewCommandNode("report", "Reporting and documentation tools")
	report.Handler = c.handleReportPlaceholder
	c.rootCommand.AddChild(report)
}

// buildSupportCommandsV2 builds support commands
func (c *ConsoleV2) buildSupportCommandsV2() {
	// support is a DIRECTORY
	support := NewCommandNode("support", "Support and attribution tools")
	support.Handler = c.handleSupportPlaceholder
	c.rootCommand.AddChild(support)
}

// buildStateCommandsV2 builds state commands
func (c *ConsoleV2) buildStateCommandsV2() {
	// state is a DIRECTORY for consciousness state management
	stateCmd := NewCommandNode("state", "Consciousness collaboration state management")
	
	// state/create command
	create := NewCommandNode("create", "Create new hybrid state package")
	create.Handler = c.handleStateCreate
	stateCmd.AddChild(create)
	
	// state/list command
	list := NewCommandNode("list", "List available state packages")
	list.Handler = c.handleStateList
	stateCmd.AddChild(list)
	
	// state/load command
	load := NewCommandNode("load", "Load a state package")
	load.Handler = c.handleStateLoad
	stateCmd.AddChild(load)
	
	// state/save command
	save := NewCommandNode("save", "Save current state")
	save.Handler = c.handleStateSave
	stateCmd.AddChild(save)
	
	c.rootCommand.AddChild(stateCmd)
}

// buildJobsCommandV2 builds jobs command
func (c *ConsoleV2) buildJobsCommandV2() {
	// jobs is a single COMMAND, not a directory
	jobs := NewCommandNode("jobs", "List running background jobs")
	jobs.Handler = c.handleJobsList
	c.rootCommand.AddChild(jobs)
}

// Placeholder handlers
func (c *ConsoleV2) handleRespondPlaceholder(console interface{}, cmd *ParsedCommand) error {
	c.Info("Respond context not yet implemented")
	return nil
}

func (c *ConsoleV2) handleReportPlaceholder(console interface{}, cmd *ParsedCommand) error {
	c.Info("Report context not yet implemented")
	return nil
}

func (c *ConsoleV2) handleSupportPlaceholder(console interface{}, cmd *ParsedCommand) error {
	c.Info("Support context not yet implemented")
	return nil
}

func (c *ConsoleV2) handleJobsList(console interface{}, cmd *ParsedCommand) error {
	c.Info("No background jobs running")
	return nil
}

// State handlers
func (c *ConsoleV2) handleStateCreate(console interface{}, cmd *ParsedCommand) error {
	c.Info("State create not yet implemented")
	return nil
}

func (c *ConsoleV2) handleStateList(console interface{}, cmd *ParsedCommand) error {
	c.Info("State list not yet implemented")
	return nil
}

func (c *ConsoleV2) handleStateLoad(console interface{}, cmd *ParsedCommand) error {
	c.Info("State load not yet implemented")
	return nil
}

func (c *ConsoleV2) handleStateSave(console interface{}, cmd *ParsedCommand) error {
	c.Info("State save not yet implemented")
	return nil
}