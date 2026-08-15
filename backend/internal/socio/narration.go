package socio

// QualitativeLabel translates an exact HP pool value into spoiler-safe,
// in-world condition language (kernel-88 spec §7): "Do not spoil hide the
// ball... communicate only what the Character could reasonably perceive."
// This is a pure ratio-threshold table -- deterministic, no LLM (product
// decision) -- so the same exact numbers always narrate the same way, and
// the wording can be swapped without touching any caller.
func QualitativeLabel(current, max int) string {
	if max <= 0 {
		return "unknown"
	}
	if current <= 0 {
		return "broken"
	}
	ratio := float64(current) / float64(max)
	switch {
	case ratio >= 0.75:
		return "steady"
	case ratio >= 0.4:
		return "strained"
	case ratio > 0:
		return "critical"
	default:
		return "broken"
	}
}
