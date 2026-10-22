package utils

import (
	"fmt"
	"os"
	"path/filepath"
)

// GetBinPath looks for a helper binary next to the running executable, in
// ../libexec/happ-tui (the installed layout; ../lib/happ-tui before v0.4), in
// ./ and ./bin (a source checkout)
// and in /usr/bin and /usr/local/bin.
func GetBinPath(name string) (string, error) {
	paths := []string{}
	if exe, err := os.Executable(); err == nil {
		if exe, err = filepath.EvalSymlinks(exe); err == nil {
			dir := filepath.Dir(exe)
			paths = append(paths,
				filepath.Join(dir, name),
				filepath.Join(dir, "..", "libexec", "happ-tui", name),
				filepath.Join(dir, "..", "lib", "happ-tui", name),
			)
		}
	}
	paths = append(paths,
		"./"+name,
		filepath.Join(".", "bin", name),
		filepath.Join("/usr/bin", name),
		filepath.Join("/usr/local/bin", name),
	)

	for _, p := range paths {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return filepath.Abs(p)
		}
	}

	return "", fmt.Errorf("binary %q not found", name)
}
