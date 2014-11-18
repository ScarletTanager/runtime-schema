package models

import (
	"encoding/json"
	"reflect"
	"regexp"
)

type DesiredLRP struct {
	ProcessGuid          string                `json:"process_guid"`
	Domain               string                `json:"domain"`
	RootFSPath           string                `json:"root_fs"`
	Instances            int                   `json:"instances"`
	Stack                string                `json:"stack"`
	EnvironmentVariables []EnvironmentVariable `json:"env,omitempty"`
	Setup                *ExecutorAction       `json:"setup,omitempty"`
	Action               ExecutorAction        `json:"action"`
	Monitor              *ExecutorAction       `json:"monitor,omitempty"`
	DiskMB               int                   `json:"disk_mb"`
	MemoryMB             int                   `json:"memory_mb"`
	CPUWeight            uint                  `json:"cpu_weight"`
	Ports                []uint32              `json:"ports"`
	Routes               []string              `json:"routes"`
	LogSource            string                `json:"log_source"`
	LogGuid              string                `json:"log_guid"`
	Annotation           string                `json:"annotation,omitempty"`
}

type DesiredLRPChange struct {
	Before *DesiredLRP
	After  *DesiredLRP
}

type DesiredLRPUpdate struct {
	Instances  *int
	Routes     []string
	Annotation *string
}

func (desired DesiredLRP) ApplyUpdate(update DesiredLRPUpdate) DesiredLRP {
	if update.Instances != nil {
		desired.Instances = *update.Instances
	}
	if update.Routes != nil {
		desired.Routes = update.Routes
	}
	if update.Annotation != nil {
		desired.Annotation = *update.Annotation
	}
	return desired
}

var processGuidPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func (desired DesiredLRP) Validate() error {
	if desired.Domain == "" {
		return ErrInvalidJSONMessage{"domain"}
	}

	if !processGuidPattern.MatchString(desired.ProcessGuid) {
		return ErrInvalidJSONMessage{"process_guid"}
	}

	if desired.Stack == "" {
		return ErrInvalidJSONMessage{"stack"}
	}

	if err := desired.Action.Validate(); err != nil {
		return err
	}

	if desired.Instances < 1 {
		return ErrInvalidJSONMessage{"instances"}
	}

	if desired.CPUWeight > 100 {
		return ErrInvalidJSONMessage{"cpu_weight"}
	}

	if len(desired.Annotation) > maximumAnnotationLength {
		return ErrInvalidJSONMessage{"annotation"}
	}

	return nil
}

func (desired DesiredLRP) ValidateModifications(updatedModel DesiredLRP) error {
	if desired.ProcessGuid != updatedModel.ProcessGuid {
		return ErrInvalidModification{"process_guid"}
	}

	if desired.Domain != updatedModel.Domain {
		return ErrInvalidModification{"domain"}
	}

	if desired.RootFSPath != updatedModel.RootFSPath {
		return ErrInvalidModification{"root_fs"}
	}

	if desired.Stack != updatedModel.Stack {
		return ErrInvalidModification{"stack"}
	}

	if !reflect.DeepEqual(desired.EnvironmentVariables, updatedModel.EnvironmentVariables) {
		return ErrInvalidModification{"env"}
	}

	if !reflect.DeepEqual(desired.Action, updatedModel.Action) {
		return ErrInvalidModification{"action"}
	}

	if desired.DiskMB != updatedModel.DiskMB {
		return ErrInvalidModification{"disk_mb"}
	}

	if desired.MemoryMB != updatedModel.MemoryMB {
		return ErrInvalidModification{"memory_mb"}
	}

	if desired.CPUWeight != updatedModel.CPUWeight {
		return ErrInvalidModification{"cpu_weight"}
	}

	if !reflect.DeepEqual(desired.Ports, updatedModel.Ports) {
		return ErrInvalidModification{"ports"}
	}

	if desired.LogSource != updatedModel.LogSource {
		return ErrInvalidModification{"log_source"}
	}

	if desired.LogGuid != updatedModel.LogGuid {
		return ErrInvalidModification{"log_guid"}
	}

	return nil
}

func NewDesiredLRPFromJSON(payload []byte) (DesiredLRP, error) {
	var lrp DesiredLRP

	err := json.Unmarshal(payload, &lrp)
	if err != nil {
		return DesiredLRP{}, err
	}

	err = lrp.Validate()
	if err != nil {
		return DesiredLRP{}, err
	}

	return lrp, nil
}

func (desired DesiredLRP) ToJSON() []byte {
	bytes, err := json.Marshal(desired)
	if err != nil {
		panic(err)
	}

	return bytes
}
