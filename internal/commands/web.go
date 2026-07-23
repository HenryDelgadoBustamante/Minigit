package commands

import (
	"fmt"
	"strconv"
	"strings"

	"minigit/internal/server"
)

// RunWeb starts the local read-only web viewer server for MiniGit scanning startDir for repositories.
func RunWeb(startDir string, portStr string) error {
	port := 8080
	portStr = strings.TrimSpace(portStr)

	if portStr != "" {
		if parsed, err := strconv.Atoi(portStr); err == nil && parsed > 0 && parsed <= 65535 {
			port = parsed
		}
	}

	if startDir == "" {
		startDir = "."
	}

	srv := server.NewServer(startDir)
	if len(srv.Projects) == 0 {
		return fmt.Errorf("no se encontraron repositorios .minigit en '%s' ni en subdirectorios (ej: proyectos_minigit/)", startDir)
	}

	return srv.StartWebServer(port)
}
