package storyboards

// Timeline built-in template (Kernel 82). Code-defined seed data, not a
// DB row (spec 4.2's explicit "may be... code-defined seed data" option)
// -- there is nothing in the database for a saved instance to reference
// back into, which is what makes "instantiate template != edit template"
// (spec 4.1) true by construction rather than by convention or access
// control. CreateTimelineBoard reads this constant once, at creation
// time, and never again; every saved Timeline is independent from that
// point on, exactly like every other Storyboard.
//
// Bumping TimelineTemplateVersion changes what future "New Timeline"
// instantiations start with. It never touches any already-created board
// -- Storyboard.TemplateVersion just records which version a given board
// was instantiated from, a historical fact.

const TimelineTemplateVersion = 1

// timelineColumnSeed pairs a default column title with its structural
// role (spec 2.2).
type timelineColumnSeed struct {
	title string
	role  string
}

var timelineDefaultColumns = []timelineColumnSeed{
	{title: "Beginning", role: ColumnRoleBeginning},
	{title: "Middle", role: ColumnRoleOrdinary},
	{title: "Ending", role: ColumnRoleEnding},
}

const timelineDefaultBandLabel = "Band 1"
const timelineDefaultRowLabel = "Row 1"

// timelineReferenceFieldSeed describes one default Reference Panel field.
// Only one of TextContent or (Items/paired) applies per FieldType; every
// seeded field starts empty -- the template supplies structure, never
// content, matching spec 3.2's "these are initial labels only."
type timelineReferenceFieldSeed struct {
	label     string
	fieldType string
	sublabelA string
	sublabelB string
}

// timelineDefaultReferenceFields seeds Premise/Beginning/Ending as plain
// long-text fields, plus a single paired_list field combining "Include"
// and "Exclude" as its two sublabels -- spec 3.2 lists five labels
// (Premise, Beginning, Ending, Include, Exclude) and spec 3.7 names
// "Include | Exclude" as its own flagship paired-list example, so this
// reads as four fields, not five: Include/Exclude are one field's two
// sides, not two separate fields. Recorded as a deliberate interpretation
// in the Kernel 82 reportback, not silently assumed.
var timelineDefaultReferenceFields = []timelineReferenceFieldSeed{
	{label: "Premise", fieldType: FieldTypeLongText},
	{label: "Beginning", fieldType: FieldTypeLongText},
	{label: "Ending", fieldType: FieldTypeLongText},
	{label: "Include / Exclude", fieldType: FieldTypePairedList, sublabelA: "Include", sublabelB: "Exclude"},
}
