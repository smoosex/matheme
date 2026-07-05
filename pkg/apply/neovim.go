package apply

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"

	"github.com/neovim/go-client/nvim"
)

func SwitchNvimThemeInGo(curTheme, initPath, chadrcPath string) error {
	content, err := os.ReadFile(chadrcPath)
	if err != nil {
		return fmt.Errorf("failed to read chadrc file '%s': %w", chadrcPath, err)
	}

	re := regexp.MustCompile(`(M\.base46\s*=\s*\{[\s\S]*?theme\s*=\s*")[^"]+(")`)
	newContent := re.ReplaceAllString(string(content), fmt.Sprintf(`${1}%s${2}`, curTheme))

	if err := os.WriteFile(chadrcPath, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("failed to write chadrc file '%s': %w", chadrcPath, err)
	}

	tmpDir := os.TempDir()
	pattern := filepath.Join(tmpDir, "nvim.*", "*", "nvim.*")
	sockets, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("error when find nvim socket: %w", err)
	}

	pidRegex := regexp.MustCompile(`nvim\.(\d+)\..*$`)
	reloadCmd := `require('nvchad.themes.utils').reload_theme(...)`
	hasActiveSocket := false

	for _, sock := range sockets {
		matches := pidRegex.FindStringSubmatch(filepath.Base(sock))
		if len(matches) < 2 {
			continue
		}

		pid, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}

		if !isProcessAlive(pid) {
			_ = os.Remove(sock)
			continue
		}

		hasActiveSocket = true

		client, err := nvim.Dial(sock)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to connect to the socket via Dial '%s': %v\n", sock, err)
			continue
		}

		if err := client.ExecLua(reloadCmd, nil, curTheme); err != nil {
			fmt.Fprintf(os.Stderr, "failed to send reload command to '%s': %v\n", sock, err)
		}

		if err := client.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "failed to close nvim client for '%s': %v\n", sock, err)
		}
	}

	if hasActiveSocket {
		return nil
	}

	if err := rebuildBase46Cache(initPath); err != nil {
		return err
	}

	return nil
}

func rebuildBase46Cache(initPath string) error {
	args := []string{"--headless"}
	if initPath != "" {
		args = append(args, "-u", initPath)
	}
	args = append(args,
		"-c", "lua require('base46').load_all_highlights()",
		"-c", "qall",
	)

	output, err := exec.Command("nvim", args...).CombinedOutput()
	if err != nil {
		trimmedOutput := strings.TrimSpace(string(output))
		if trimmedOutput != "" {
			return fmt.Errorf("failed to rebuild neovim base46 cache: %w: %s", err, trimmedOutput)
		}
		return fmt.Errorf("failed to rebuild neovim base46 cache: %w", err)
	}

	return nil
}

func isProcessAlive(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	err = process.Signal(syscall.Signal(0))
	return err == nil
}
