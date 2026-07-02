package characters

// Chapter4CourtyardScene is the canonical, static initial Locked Courtyard
// scene projection. Per Kernel 56 section 7 (Event C4-E10), this establishes
// only the required scene facts -- it does not invent Kessa's dialogue,
// resolve the locked door, or script any encounter/confrontation mechanics.
type Chapter4CourtyardScene struct {
	SceneID     string   `json:"scene_id"`
	Title       string   `json:"title"`
	Description []string `json:"description"`
	NPCs        []string `json:"npcs"`
	DoorState   string   `json:"door_state"`
}

var chapter4CourtyardScene = Chapter4CourtyardScene{
	SceneID: "locked_courtyard_v1",
	Title:   "The Locked Courtyard",
	Description: []string{
		"Ancient stone walls rise high on every side, enclosing a wide courtyard worn smooth by countless feet.",
		"Market booths and stalls crowd the open ground, their keepers calling out over a dense, restless crowd of voices and motion.",
		"At the far end stands a massive ironbound oak door -- shut fast, its lock unmoved by anything the crowd has tried.",
	},
	NPCs:      []string{"Kessa"},
	DoorState: "locked",
}

// LoadChapter4CourtyardScene returns the static initial Courtyard scene
// projection. It is deliberately data-only: no dialogue, no door resolution,
// no encounter state.
func LoadChapter4CourtyardScene() Chapter4CourtyardScene {
	return chapter4CourtyardScene
}
