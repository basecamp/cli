package profile

import (
	"fmt"
	"sort"
)

// ResolveOptions controls how profile resolution behaves.
type ResolveOptions struct {
	// FlagValue is the --profile flag value (highest priority).
	FlagValue string

	// EnvVar is the environment variable value (e.g., APP_PROFILE).
	EnvVar string

	// DefaultProfile is the default_profile from config.
	DefaultProfile string

	// Profiles is the set of known profiles.
	Profiles map[string]*Profile

	// Interactive is true when the user can be prompted (TTY, no --agent/--json).
	// When true and multiple profiles exist with no selection, Picker is called.
	Interactive bool

	// Picker prompts the user to choose a profile. Only called when
	// Interactive is true and multiple profiles exist with no other selection.
	// Returns the chosen profile name. If nil, resolution fails instead of prompting.
	Picker func(names []string) (string, error)
}

// Resolve determines the active profile using strict precedence:
//
//  1. --profile flag
//  2. APP_PROFILE env var
//  3. default_profile in config
//  4. Auto-select if exactly one profile exists
//  5. Interactive picker (if available)
//  6. Error
//
// Returns ("", nil) when no profiles are configured (profile-less mode).
func Resolve(opts ResolveOptions) (string, error) {
	if len(opts.Profiles) == 0 {
		return "", nil
	}

	// 1. Flag
	if opts.FlagValue != "" {
		if _, ok := opts.Profiles[opts.FlagValue]; !ok {
			return "", fmt.Errorf("profile %q not found", opts.FlagValue)
		}
		return opts.FlagValue, nil
	}

	// 2. Env var
	if opts.EnvVar != "" {
		if _, ok := opts.Profiles[opts.EnvVar]; !ok {
			return "", fmt.Errorf("profile %q (from environment) not found", opts.EnvVar)
		}
		return opts.EnvVar, nil
	}

	// 3. Config default
	if opts.DefaultProfile != "" {
		if _, ok := opts.Profiles[opts.DefaultProfile]; !ok {
			return "", fmt.Errorf("default profile %q not found", opts.DefaultProfile)
		}
		return opts.DefaultProfile, nil
	}

	// 4. Auto-select single profile
	if len(opts.Profiles) == 1 {
		for name := range opts.Profiles {
			return name, nil
		}
	}

	// 5. Interactive picker
	if opts.Interactive && opts.Picker != nil {
		names := make([]string, 0, len(opts.Profiles))
		for name := range opts.Profiles {
			names = append(names, name)
		}
		sort.Strings(names)
		return opts.Picker(names)
	}

	// 6. Error — multiple profiles, no selection
	return "", fmt.Errorf("multiple profiles configured; use --profile or set a default")
}
