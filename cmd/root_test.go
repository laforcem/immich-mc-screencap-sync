//go:build windows

package cmd

import (
	"testing"
)

func TestApplyMousetrap(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		fromExplorer bool
		want        []string
	}{
		{
			name:        "explorer launch injects tray",
			args:        []string{"screenshot-sync.exe"},
			fromExplorer: true,
			want:        []string{"screenshot-sync.exe", "tray"},
		},
		{
			name:        "explorer launch preserves extra args",
			args:        []string{"screenshot-sync.exe", "--config", "foo.toml"},
			fromExplorer: true,
			want:        []string{"screenshot-sync.exe", "tray", "--config", "foo.toml"},
		},
		{
			name:        "terminal launch unchanged with no args",
			args:        []string{"screenshot-sync.exe"},
			fromExplorer: false,
			want:        []string{"screenshot-sync.exe"},
		},
		{
			name:        "terminal launch unchanged with subcommand",
			args:        []string{"screenshot-sync.exe", "sync"},
			fromExplorer: false,
			want:        []string{"screenshot-sync.exe", "sync"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := applyMousetrap(tt.args, tt.fromExplorer)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("got %v, want %v", got, tt.want)
				}
			}
		})
	}
}
