package minecraft

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ReadPrismAccount reads the active Minecraft username from Prism Launcher's accounts.json.
func ReadPrismAccount(prismDir string) (string, error) {
	path := filepath.Join(prismDir, "accounts.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	var f prismAccountsFile
	if err := json.Unmarshal(data, &f); err != nil {
		return "", fmt.Errorf("parse %s: %w", path, err)
	}
	name := f.activeName()
	if name == "" {
		return "", fmt.Errorf("no account name found in %s", path)
	}
	return name, nil
}

// ReadVanillaAccount reads the active Minecraft username from the vanilla launcher.
// Tries launcher_accounts.json (newer format) then launcher_profiles.json (older format).
func ReadVanillaAccount() (string, error) {
	appdata := os.Getenv("APPDATA")
	if appdata == "" {
		return "", fmt.Errorf("APPDATA not set")
	}
	mcDir := filepath.Join(appdata, ".minecraft")

	if name, err := readLauncherAccounts(filepath.Join(mcDir, "launcher_accounts.json")); err == nil {
		return name, nil
	}
	return readLauncherProfiles(filepath.Join(mcDir, "launcher_profiles.json"))
}

// prism format
type prismAccountsFile struct {
	Accounts             []prismAccount `json:"accounts"`
	ActiveAccountLocalID string         `json:"activeAccountLocalId"`
}

type prismAccount struct {
	LocalID string       `json:"localId"`
	Profile prismProfile `json:"profile"`
}

type prismProfile struct {
	Name string `json:"name"`
}

func (f *prismAccountsFile) activeName() string {
	for _, acc := range f.Accounts {
		if acc.LocalID == f.ActiveAccountLocalID && acc.Profile.Name != "" {
			return acc.Profile.Name
		}
	}
	for _, acc := range f.Accounts {
		if acc.Profile.Name != "" {
			return acc.Profile.Name
		}
	}
	return ""
}

// newer vanilla format (launcher_accounts.json, Microsoft auth)
type launcherAccountsFile struct {
	Accounts             map[string]launcherAccount `json:"accounts"`
	ActiveAccountLocalID string                     `json:"activeAccountLocalId"`
}

type launcherAccount struct {
	MinecraftProfile struct {
		Name string `json:"name"`
	} `json:"minecraftProfile"`
}

func readLauncherAccounts(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var f launcherAccountsFile
	if err := json.Unmarshal(data, &f); err != nil {
		return "", err
	}
	if acc, ok := f.Accounts[f.ActiveAccountLocalID]; ok && acc.MinecraftProfile.Name != "" {
		return acc.MinecraftProfile.Name, nil
	}
	for _, acc := range f.Accounts {
		if acc.MinecraftProfile.Name != "" {
			return acc.MinecraftProfile.Name, nil
		}
	}
	return "", fmt.Errorf("no account in %s", path)
}

// older vanilla format (launcher_profiles.json)
type launcherProfilesFile struct {
	AuthenticationDatabase map[string]struct {
		Profiles map[string]struct {
			DisplayName string `json:"displayName"`
		} `json:"profiles"`
	} `json:"authenticationDatabase"`
	SelectedUser struct {
		Account string `json:"account"`
		Profile string `json:"profile"`
	} `json:"selectedUser"`
}

func readLauncherProfiles(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var f launcherProfilesFile
	if err := json.Unmarshal(data, &f); err != nil {
		return "", err
	}
	acc, ok := f.AuthenticationDatabase[f.SelectedUser.Account]
	if !ok {
		return "", fmt.Errorf("selected account not found in %s", path)
	}
	profile, ok := acc.Profiles[f.SelectedUser.Profile]
	if !ok || profile.DisplayName == "" {
		return "", fmt.Errorf("selected profile not found in %s", path)
	}
	return profile.DisplayName, nil
}
