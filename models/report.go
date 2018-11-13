package models

type Report struct {
	ModuleName      string       `json:"moduleName"`
	PartName        string       `json:"partName"`
	LutSummary      GroupSummary `json:"lutSummary"`
	RegSummary      GroupSummary `json:"regSummary"`
	BlockRamSummary GroupSummary `json:"blockRamSummary"`
	UltraRamSummary PartDetail   `json:"ultraRamSummary"`
	DspBlockSummary PartDetail   `json:"dspBlockSummary"`
	WeightedAverage PartDetail   `json:"weightedAverage"`
}

type GroupSummary struct {
	Description string      `json:"description"`
	Used        int         `json:"used"`
	Available   int         `json:"available"`
	Utilisation float32     `json:"utilisation"`
	Detail      PartDetails `json:"detail"`
}

type PartDetails map[string]PartDetail

type PartDetail struct {
	Description string  `json:"description"`
	Used        int     `json:"used"`
	Available   int     `json:"available"`
	Utilisation float32 `json:"utilisation"`
}
