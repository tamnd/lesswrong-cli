package cli

import (
	"errors"

	"github.com/tamnd/lesswrong-cli/lesswrong"
)

func isNotFound(err error) bool {
	return errors.Is(err, lesswrong.ErrNotFound)
}
