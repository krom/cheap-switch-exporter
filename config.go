package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// RootConfig is the top-level shape of config.yaml: a list of named profiles plus
// optional global settings applied to any profile that doesn't set its own value.
type RootConfig struct {
	Profiles []map[string]ProfileConfig `yaml:"profiles"`
	Settings GlobalSettings             `yaml:"settings"`
}

// GlobalSettings provides defaults for poll_rate_seconds/timeout_seconds, applied to any
// profile that doesn't set its own value (see resolveProfileConfig).
type GlobalSettings struct {
	PollRateSeconds int `yaml:"poll_rate_seconds"`
	TimeoutSeconds  int `yaml:"timeout_seconds"`
}

// ProfileConfig holds one switch's connection details, dialect selection, and per-port
// comment labels, as configured under a single entry of config.yaml's profiles list.
type ProfileConfig struct {
	Profile         string            `yaml:"profile"`
	Address         string            `yaml:"address"`
	Username        string            `yaml:"username"`
	Password        string            `yaml:"password"`
	Timeout         int               `yaml:"timeout_seconds"`
	PollRateSeconds int               `yaml:"poll_rate_seconds"`
	PoE             int               `yaml:"poe"`
	Comments        map[string]string `yaml:"comments"`
}

// NamedProfile pairs a profile's config.yaml key (used as the "switch" metric label)
// with its resolved ProfileConfig.
type NamedProfile struct {
	Name   string
	Config ProfileConfig
}

// resolveProfileConfig fills in a profile's Timeout/PollRateSeconds using the precedence
// profile value -> global settings value -> hardcoded default (10s timeout, 60s poll rate).
func resolveProfileConfig(pc ProfileConfig, settings GlobalSettings) ProfileConfig {
	if pc.Timeout == 0 {
		if settings.TimeoutSeconds != 0 {
			pc.Timeout = settings.TimeoutSeconds
		} else {
			pc.Timeout = 10
		}
	}
	if pc.PollRateSeconds == 0 {
		if settings.PollRateSeconds != 0 {
			pc.PollRateSeconds = settings.PollRateSeconds
		} else {
			pc.PollRateSeconds = 60
		}
	}
	return pc
}

// validateProfileConfig checks a resolved profile's Profile/PoE combination is valid,
// returning a descriptive error naming the profile if not.
func validateProfileConfig(name string, pc ProfileConfig) error {
	switch pc.Profile {
	case "", "default", "v2":
	default:
		return fmt.Errorf("profile %q: invalid profile %q (must be \"\", \"default\", or \"v2\")", name, pc.Profile)
	}
	if pc.Profile == "v2" && pc.PoE == 1 {
		return fmt.Errorf("profile %q: poe: 1 is not supported with profile: v2", name)
	}
	return nil
}

// loadConfig reads and parses path as a RootConfig, then resolves and validates every
// configured profile, returning the flattened, ready-to-use profile list.
func loadConfig(path string) ([]NamedProfile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg RootConfig
	if err := yaml.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	var profiles []NamedProfile
	for _, m := range cfg.Profiles {
		for name, pc := range m {
			rc := resolveProfileConfig(pc, cfg.Settings)
			if err := validateProfileConfig(name, rc); err != nil {
				return nil, err
			}
			profiles = append(profiles, NamedProfile{Name: name, Config: rc})
		}
	}
	return profiles, nil
}
