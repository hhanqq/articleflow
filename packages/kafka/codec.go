package kafka

import "encoding/json"

func MarshalJSON(value any) ([]byte, error) {
	return json.Marshal(value)
}

func UnmarshalJSON(payload []byte, target any) error {
	return json.Unmarshal(payload, target)
}

