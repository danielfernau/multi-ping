package app

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"sort"
	"strings"
	"time"

	"multi-ping/internal/ping"
)

type Config struct {
	Checks   []ping.CheckKey
	Interval time.Duration
	Timeout  time.Duration
}

type checkList []string

func (c *checkList) String() string {
	return strings.Join(*c, ",")
}

func (c *checkList) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return errors.New("empty check value")
	}
	*c = append(*c, value)
	return nil
}

func ParseConfig(args []string, stderr io.Writer) (Config, error) {
	var checks checkList
	var interfacesCSV string
	var targetsCSV string
	var interval time.Duration
	var timeout time.Duration

	fs := flag.NewFlagSet("multi-ping", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Var(&checks, "check", "Interface-specific target mapping in the form iface=ip1,ip2 (repeatable)")
	fs.StringVar(&interfacesCSV, "interfaces", "", "Comma-separated interfaces to probe")
	fs.StringVar(&targetsCSV, "targets", "", "Comma-separated targets to probe through every interface")
	fs.DurationVar(&interval, "interval", 2*time.Second, "Delay between probe cycles")
	fs.DurationVar(&timeout, "timeout", 1500*time.Millisecond, "Per-ping timeout")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "Usage:")
		fmt.Fprintln(stderr, "  multi-ping -interfaces eth0,wlan0 -targets 8.8.8.8,1.1.1.1")
		fmt.Fprintln(stderr, "  multi-ping -check eth0=8.8.8.8,1.1.1.1 -check tun0=10.0.0.1")
		fmt.Fprintln(stderr)
		fmt.Fprintln(stderr, "Flags:")
		fs.PrintDefaults()
	}

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if interval <= 0 {
		return Config{}, errors.New("interval must be greater than zero")
	}
	if timeout <= 0 {
		return Config{}, errors.New("timeout must be greater than zero")
	}

	parsedChecks, err := collectChecks(checks, interfacesCSV, targetsCSV)
	if err != nil {
		return Config{}, err
	}

	if err := validateInterfaces(parsedChecks); err != nil {
		return Config{}, err
	}

	sort.Slice(parsedChecks, func(i, j int) bool {
		if parsedChecks[i].Interface == parsedChecks[j].Interface {
			return parsedChecks[i].Target < parsedChecks[j].Target
		}
		return parsedChecks[i].Interface < parsedChecks[j].Interface
	})

	return Config{
		Checks:   dedupeChecks(parsedChecks),
		Interval: interval,
		Timeout:  timeout,
	}, nil
}

func collectChecks(explicit checkList, interfacesCSV, targetsCSV string) ([]ping.CheckKey, error) {
	if len(explicit) > 0 {
		var checks []ping.CheckKey
		for _, raw := range explicit {
			parsed, err := parseCheck(raw)
			if err != nil {
				return nil, err
			}
			checks = append(checks, parsed...)
		}
		if len(checks) == 0 {
			return nil, errors.New("no valid checks were supplied")
		}
		return checks, nil
	}

	interfaces := splitCSV(interfacesCSV)
	targets := splitCSV(targetsCSV)

	if len(interfaces) == 0 || len(targets) == 0 {
		return nil, errors.New("provide either one or more -check flags, or both -interfaces and -targets")
	}

	var checks []ping.CheckKey
	for _, iface := range interfaces {
		for _, target := range targets {
			checks = append(checks, ping.CheckKey{
				Interface: iface,
				Target:    target,
			})
		}
	}

	return checks, nil
}

func parseCheck(raw string) ([]ping.CheckKey, error) {
	parts := strings.SplitN(raw, "=", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid -check %q: expected iface=ip1,ip2", raw)
	}

	iface := strings.TrimSpace(parts[0])
	targets := splitCSV(parts[1])
	if iface == "" || len(targets) == 0 {
		return nil, fmt.Errorf("invalid -check %q: interface and at least one target are required", raw)
	}

	var checks []ping.CheckKey
	for _, target := range targets {
		checks = append(checks, ping.CheckKey{
			Interface: iface,
			Target:    target,
		})
	}
	return checks, nil
}

func splitCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		values = append(values, part)
	}
	return values
}

func validateInterfaces(checks []ping.CheckKey) error {
	seen := map[string]struct{}{}
	for _, check := range checks {
		if check.Target == "" {
			return fmt.Errorf("empty target for interface %q", check.Interface)
		}
		if _, ok := seen[check.Interface]; ok {
			continue
		}
		if _, err := net.InterfaceByName(check.Interface); err != nil {
			return fmt.Errorf("unknown interface %q", check.Interface)
		}
		seen[check.Interface] = struct{}{}
	}
	return nil
}

func dedupeChecks(checks []ping.CheckKey) []ping.CheckKey {
	seen := make(map[ping.CheckKey]struct{}, len(checks))
	deduped := make([]ping.CheckKey, 0, len(checks))
	for _, check := range checks {
		if _, ok := seen[check]; ok {
			continue
		}
		seen[check] = struct{}{}
		deduped = append(deduped, check)
	}
	return deduped
}
