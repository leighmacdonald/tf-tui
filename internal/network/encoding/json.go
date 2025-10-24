package encoding

import (
	"encoding/json"
	"errors"
	"io"
)

var ErrDecodeJSON = errors.New("failed to decode JSON")

func UnmarshalJSON[T any](reader io.Reader) (T, error) {
	var value T
	if err := json.NewDecoder(reader).Decode(&value); err != nil {
		return value, errors.Join(err, ErrDecodeJSON)
	}

	return value, nil
}
