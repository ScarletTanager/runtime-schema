package models

type DesireAppRequestFromCC struct {
	AppId        string                `json:"app_id"`
	AppVersion   string                `json:"app_version"`
	DropletUri   string                `json:"droplet_uri"`
	StartCommand string                `json:"start_command"`
	Environment  []EnvironmentVariable `json:"environment"`
}
