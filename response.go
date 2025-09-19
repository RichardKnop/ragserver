package ragserver

type TextValue string

type MetricValue struct {
	Value float64
	Unit  string
}

type BooleanValue bool

type Response struct {
	Text      TextValue    `json:"text"`
	Metric    MetricValue  `json:"metric"`
	Boolean   BooleanValue `json:"boolean"`
	Documents []Document   `json:"documents"`
}
