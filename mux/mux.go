package mux

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	socketDir = ".ssh/sockets"
	muxConf   = ".ssh/mux.conf"
	sshConfig = ".ssh/config"
)

var muxConfContent = `Host *
    ControlMaster auto
    ControlPath ~/.ssh/sockets/%r@%h-%p
`

func home() (string, error) {
	h, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not determine home directory: %w", err)
	}
	return h, nil
}

func checkInitialized() error {
	h, err := home()
	if err != nil {
		return err
	}
	if _, err := os.Stat(filepath.Join(h, muxConf)); os.IsNotExist(err) {
		return fmt.Errorf("ssh-mux not initialized — run 'ssh-mux init' first")
	}

	out, err := exec.Command("ssh", "-G", "github.com").Output()
	if err != nil {
		return fmt.Errorf("ssh config is invalid: %w", err)
	}
	if !strings.Contains(string(out), filepath.Join(h, socketDir)) {
		return fmt.Errorf("mux.conf is not included in your SSH config — add 'Include ~/.ssh/mux.conf' to ~/.ssh/config")
	}
	return nil
}

func socketPath(h, dest string, port int) (string, error) {
	user, host, err := parseDestination(dest)
	if err != nil {
		return "", err
	}
	return filepath.Join(h, socketDir, fmt.Sprintf("%s@%s-%d", user, host, port)), nil
}

func parseDestination(dest string) (user, host string, err error) {
	if before, after, ok := strings.Cut(dest, "@"); ok {
		return before, after, nil
	}
	return "", "", fmt.Errorf("destination must be in user@host format, got %q", dest)
}

func Init(force, noEdit bool) error {
	h, err := home()
	if err != nil {
		return err
	}

	sockDir := filepath.Join(h, socketDir)
	if err := os.MkdirAll(sockDir, 0700); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	muxPath := filepath.Join(h, muxConf)
	if _, err := os.Stat(muxPath); err == nil && !force {
		return fmt.Errorf("%s already exists (use --force to overwrite)", muxPath)
	}

	if err := os.WriteFile(muxPath, []byte(muxConfContent), 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", muxPath, err)
	}
	fmt.Printf("wrote %s\n", muxPath)

	if !noEdit {
		cfgPath := filepath.Join(h, sshConfig)
		includeLine := "Include ~/.ssh/mux.conf"

		cfgBytes, err := os.ReadFile(cfgPath)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to read %s: %w", cfgPath, err)
		}

		if strings.Contains(string(cfgBytes), includeLine) {
			fmt.Printf("%s already includes mux.conf\n", cfgPath)
		} else {
			backupPath := fmt.Sprintf("%s.bak.%d", cfgPath, time.Now().Unix())
			if len(cfgBytes) > 0 {
				if err := os.WriteFile(backupPath, cfgBytes, 0644); err != nil {
					return fmt.Errorf("failed to back up %s: %w", cfgPath, err)
				}
				fmt.Printf("backed up %s → %s\n", cfgPath, backupPath)
			}

			newContent := includeLine + "\n\n" + string(cfgBytes)
			if err := os.WriteFile(cfgPath, []byte(newContent), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", cfgPath, err)
			}
			fmt.Printf("added Include directive to %s\n", cfgPath)
		}

		cmd := exec.Command("ssh", "-G", "github.com")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("ssh config validation failed — check %s manually: %w", cfgPath, err)
		}
	} else {
		fmt.Println("skipped ~/.ssh/config (--no-edit)")
		fmt.Println("ensure your SSH config includes the following:")
		fmt.Println()
		fmt.Println("  Include ~/.ssh/mux.conf")
	}

	return nil
}

func Up(dest string, port int) error {
	if err := checkInitialized(); err != nil {
		return err
	}

	h, err := home()
	if err != nil {
		return err
	}

	sock, err := socketPath(h, dest, port)
	if err != nil {
		return err
	}

	check := exec.Command("ssh", "-S", sock, "-O", "check", dest)
	if check.Run() == nil {
		fmt.Printf("master already active for %s (socket: %s)\n", dest, sock)
		return nil
	}

	args := []string{"-M", "-S", sock, "-N", "-f"}
	if port != 22 {
		args = append(args, "-p", fmt.Sprintf("%d", port))
	}
	args = append(args, dest)

	cmd := exec.Command("ssh", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to establish ControlMaster for %s: %w", dest, err)
	}

	fmt.Printf("master connection established for %s (socket: %s)\n", dest, sock)
	return nil
}

func Down(dest string) error {
	h, err := home()
	if err != nil {
		return err
	}

	if dest != "" {
		return closeSocket(h, dest)
	}

	sockDir := filepath.Join(h, socketDir)
	entries, err := os.ReadDir(sockDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read socket directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		d, port := socketNameToDestination(name)
		if d == "" {
			continue
		}
		sock := filepath.Join(sockDir, name)
		cmd := exec.Command("ssh", "-S", sock, "-O", "exit", "-p", fmt.Sprintf("%d", port), d)
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close %s: %v\n", name, err)
		} else {
			fmt.Printf("closed %s\n", d)
		}
	}
	return nil
}

func closeSocket(h, dest string) error {
	sock, err := socketPath(h, dest, 22)
	if err != nil {
		return err
	}
	cmd := exec.Command("ssh", "-S", sock, "-O", "exit", dest)
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to close master for %s (may already be gone): %v\n", dest, err)
	} else {
		fmt.Printf("closed %s\n", dest)
	}
	return nil
}

func Status(dest string) error {
	h, err := home()
	if err != nil {
		return err
	}

	if dest != "" {
		sock, err := socketPath(h, dest, 22)
		if err != nil {
			return err
		}
		cmd := exec.Command("ssh", "-S", sock, "-O", "check", dest)
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Println("inactive")
		} else {
			fmt.Println("active")
		}
		return nil
	}

	sockDir := filepath.Join(h, socketDir)
	entries, err := os.ReadDir(sockDir)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("no sockets directory — run 'ssh-mux init' first")
			return nil
		}
		return fmt.Errorf("failed to read socket directory: %w", err)
	}

	found := false
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		d, port := socketNameToDestination(name)
		if d == "" {
			continue
		}
		sock := filepath.Join(sockDir, name)
		cmd := exec.Command("ssh", "-S", sock, "-O", "check", "-p", fmt.Sprintf("%d", port), d)
		status := "inactive"
		if cmd.Run() == nil {
			status = "active"
			found = true
		} else {
			continue
		}
		fmt.Printf("%-30s %s  (socket: %s)\n", d, status, sock)
	}

	if !found {
		fmt.Println("no active masters")
	}

	return nil
}

// socketNameToDestination reverses the %r@%h-%p pattern.
// e.g. "git@github.com-22" → ("git@github.com", 22)
func socketNameToDestination(name string) (string, int) {
	lastDash := strings.LastIndex(name, "-")
	if lastDash == -1 {
		return "", 0
	}
	var port int
	if _, err := fmt.Sscanf(name[lastDash+1:], "%d", &port); err != nil {
		return "", 0
	}
	return name[:lastDash], port
}
