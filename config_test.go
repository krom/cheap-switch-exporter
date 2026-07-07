package main

import "testing"

func TestValidateProfileConfig(t *testing.T) {
	cases := []struct {
		name    string
		pc      ProfileConfig
		wantErr bool
	}{
		{"empty profile ok", ProfileConfig{Profile: ""}, false},
		{"default profile ok", ProfileConfig{Profile: "default"}, false},
		{"v2 profile ok", ProfileConfig{Profile: "v2"}, false},
		{"invalid profile rejected", ProfileConfig{Profile: "bogus"}, true},
		{"v2 with poe rejected", ProfileConfig{Profile: "v2", PoE: 1}, true},
		{"default with poe ok", ProfileConfig{Profile: "default", PoE: 1}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			err := validateProfileConfig("test-profile", c.pc)
			if (err != nil) != c.wantErr {
				t.Errorf("validateProfileConfig(%+v) error = %v, wantErr %v", c.pc, err, c.wantErr)
			}
		})
	}
}

func TestResolveProfileConfig(t *testing.T) {
	cases := []struct {
		name         string
		pc           ProfileConfig
		settings     GlobalSettings
		wantTimeout  int
		wantPollRate int
	}{
		{
			name:         "nothing set uses hardcoded defaults",
			pc:           ProfileConfig{},
			settings:     GlobalSettings{},
			wantTimeout:  10,
			wantPollRate: 60,
		},
		{
			name:         "global settings used when profile unset",
			pc:           ProfileConfig{},
			settings:     GlobalSettings{PollRateSeconds: 30, TimeoutSeconds: 8},
			wantTimeout:  8,
			wantPollRate: 30,
		},
		{
			name:         "profile value takes precedence over global settings",
			pc:           ProfileConfig{Timeout: 20, PollRateSeconds: 15},
			settings:     GlobalSettings{PollRateSeconds: 30, TimeoutSeconds: 8},
			wantTimeout:  20,
			wantPollRate: 15,
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := resolveProfileConfig(c.pc, c.settings)
			if got.Timeout != c.wantTimeout {
				t.Errorf("Timeout = %d, want %d", got.Timeout, c.wantTimeout)
			}
			if got.PollRateSeconds != c.wantPollRate {
				t.Errorf("PollRateSeconds = %d, want %d", got.PollRateSeconds, c.wantPollRate)
			}
		})
	}
}
