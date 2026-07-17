package marketplace

import "encoding/json"

func jsonValid(raw []byte) bool {
	return json.Valid(raw)
}
