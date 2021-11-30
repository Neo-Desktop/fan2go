package configuration

type CurveConfig struct {
	ID       string               `json:"id"`
	Linear   *LinearCurveConfig   `json:"linear,omitempty"`
	Function *FunctionCurveConfig `json:"function,omitempty"`
}

type LinearCurveConfig struct {
	Sensor string      `json:"sensor"`
	Min    int         `json:"min"`
	Max    int         `json:"max"`
	Steps  map[int]int `json:"steps"`
}

const (
	FunctionAverage = "average"
	FunctionMinimum = "minimum"
	FunctionMaximum = "maximum"
)

type FunctionCurveConfig struct {
	Type   string   `json:"type"`
	Curves []string `json:"curves"`
}