package curves

import (
	"fmt"
	"github.com/markusressel/fan2go/internal/configuration"
	"github.com/markusressel/fan2go/internal/util"
)

type SpeedCurve interface {
	GetId() string
	// Evaluate calculates the current value of the given curve,
	// returns a value in [0..255]
	Evaluate() (value int, err error)
}

var (
	SpeedCurveMap = map[string]SpeedCurve{}
)

func NewSpeedCurve(config configuration.CurveConfig) (SpeedCurve, error) {
	if config.Linear != nil {
		return &linearSpeedCurve{
			ID:       config.ID,
			sensorId: config.Linear.Sensor,
			min:      config.Linear.Min,
			max:      config.Linear.Max,
			steps:    config.Linear.Steps,
		}, nil
	}

	if config.PID != nil {
		pidLoop := util.NewPidLoop(
			config.PID.Kp,
			config.PID.Ki,
			config.PID.Kd,
		)
		return &pidSpeedCurve{
			ID:       config.ID,
			sensorId: config.PID.Sensor,
			setPoint: config.PID.SetPoint,
			pidLoop:  pidLoop,
		}, nil
	}

	if config.Function != nil {
		return &functionSpeedCurve{
			ID:       config.ID,
			function: config.Function.Type,
			curveIds: config.Function.Curves,
		}, nil
	}

	return nil, fmt.Errorf("no matching curve type for curve: %s", config.ID)
}
