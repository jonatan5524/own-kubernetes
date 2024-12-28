package rest

import (
	"strings"

	"github.com/tidwall/gjson"
)

func validateFieldSelector(fieldSelector string, value string) bool {
	splitedFieldSelector := strings.Split(fieldSelector, "=")
	resGJSON := gjson.Get(string(value), splitedFieldSelector[0])

	return resGJSON.Exists() && resGJSON.Value() == splitedFieldSelector[1]
}
