package protocol

type PatchEnvelope struct {
	Sequence uint64 `json:"seq"`
	EventID  int64  `json:"eventId"`
	Type     string `json:"type"`
	Payload  any    `json:"payload"`
}

type VariablesChanged struct {
	Entries map[string]any `json:"entries"`
}
