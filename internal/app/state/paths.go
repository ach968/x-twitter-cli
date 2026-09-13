package state

import "path/filepath"

type StatePaths struct {
	ActiveContractPath    string
	CandidateContractPath string
	AuthenticationPath    string
	ProfilePath           string
	EvidenceDirectory     string
}

func ResolvePaths(homeDirectory string, environment map[string]string) StatePaths {
	configRoot := environment["XDG_CONFIG_HOME"]
	if configRoot == "" {
		configRoot = filepath.Join(homeDirectory, ".config")
	}
	stateRoot := environment["XDG_STATE_HOME"]
	if stateRoot == "" {
		stateRoot = filepath.Join(homeDirectory, ".local", "state")
	}
	configDirectory := filepath.Join(configRoot, "x-twitter-cli")
	stateDirectory := filepath.Join(stateRoot, "x-twitter-cli")
	return StatePaths{
		ActiveContractPath:    filepath.Join(configDirectory, "contracts.json"),
		CandidateContractPath: filepath.Join(configDirectory, "contracts.candidate.json"),
		AuthenticationPath:    filepath.Join(configDirectory, "authentication.json"),
		ProfilePath:           filepath.Join(stateDirectory, "chromium-profile"),
		EvidenceDirectory:     filepath.Join(stateDirectory, "evidence"),
	}
}

func ResolveContractSource(option string, environment map[string]string, defaultPath string) string {
	if option != "" {
		return option
	}
	if value := environment["TWT_CONTRACT_FILE"]; value != "" {
		return value
	}
	return defaultPath
}
