package pkg

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/google/uuid"
)

func GenerateNewID(name string) string {
	id := uuid.New()

	return fmt.Sprintf("%s-%s", name, id)
}

func ExecuteCommand(command string, waitToComplete bool) error {
	log.Printf("Executing: %s", command)

	splitedCommand := strings.Split(command, " ")
	cmd := exec.Command(splitedCommand[0], splitedCommand[1:]...)
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		return err
	}

	if waitToComplete {
		if err := cmd.Wait(); err != nil {
			log.Printf("output: %s", cmd.Stdout)

			return err
		}
	}

	log.Printf("output: %s", cmd.Stdout)

	return nil
}

func ReplaceAtIndex(str string, replacement rune, index int) string {
	return str[:index] + string(replacement) + str[index+1:]
}
