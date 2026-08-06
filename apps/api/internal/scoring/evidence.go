package scoring

// Evidence is a single scored check produced while computing a trust score.
// It explains how the final score was reached. Result is one of:
// "pass", "warning", "fail", or "info".
type Evidence struct {
	Label  string `json:"label"`
	Result string `json:"result"`
	Points int    `json:"points"`
}
