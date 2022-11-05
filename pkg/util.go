package pkg

import (
	"fmt"
	"log"
	"os"
	"os/exec"

	"github.com/google/uuid"
)

func GenerateNewID(name string) string {
	id := uuid.New()

	return fmt.Sprintf("%s-%s", name, id)
}

func ExecuteCommand(command string) error {
	cmd := exec.Command("bash", "-c", command)
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		return err
	}

	log.Printf("%s logs: %s", command, cmd.Stdout)

	return nil
}

func ReplaceAtIndex(str string, replacement rune, index int) string {
	return str[:index] + string(replacement) + str[index+1:]
}
