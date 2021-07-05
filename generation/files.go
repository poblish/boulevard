package generation

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

var userHome, _ = os.UserHomeDir()
var userDir, _ = os.Getwd()

func FriendlyFileName(path string) string {
	abs, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("Couldn't get absolute path %s", err)
	}

	return strings.Replace(strings.Replace(abs, userDir, ".", 1), userHome, "~", 1)
}
