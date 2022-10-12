package pkg

import (
	"fmt"

	"github.com/google/uuid"
)

func GenerateNewID(name string) string {
	id := uuid.New()

	return fmt.Sprintf("%s-%s", name, id)
}
