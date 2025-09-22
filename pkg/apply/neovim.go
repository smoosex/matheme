package apply

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"syscall"

	"github.com/neovim/go-client/nvim"
)

func SwitchNvimThemeInGo(curTheme, chadrcPath string) error {
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

	if len(sockets) == 0 {
		return nil
	}

	pidRegex := regexp.MustCompile(`nvim\.(\d+)\..*$`)
	reloadCmd := `require('nvchad.themes.utils').reload_theme(...)`

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

		client, err := nvim.Dial(sock)
		if err != nil {
			log.Printf("failed to connect to the socket via Dial '%s': %v", sock, err)
		}
		defer client.Close()

		if err := client.ExecLua(reloadCmd, nil, curTheme); err != nil {
			log.Printf("failed to send reload command to '%s': %v", sock, err)
		}
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
