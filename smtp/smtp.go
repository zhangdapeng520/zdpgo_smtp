package smtp

import (
	"errors"
	"strings"
)

// validateLine 校验命令行，是否有 CR or LF as per RFC 5321
func validateLine(line string) error {
	if strings.ContainsAny(line, "\n\r") {
		return errors.New("smtp: 命令行不包含 CR or LF")
	}
	return nil
}
