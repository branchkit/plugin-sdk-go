package shared

// AudioDevice describes an audio input or output device.
type AudioDevice struct {
	ID              uint32 `json:"id"`
	UID             string `json:"uid"`
	Name            string `json:"name"`
	IsInput         bool   `json:"is_input"`
	IsOutput        bool   `json:"is_output"`
	IsDefaultInput  bool   `json:"is_default_input"`
	IsDefaultOutput bool   `json:"is_default_output"`
}

// AudioDeviceList is the response from the audio-devices endpoint.
type AudioDeviceList struct {
	Devices []AudioDevice `json:"devices"`
}

// ApplescriptResult is the response from run-applescript.
type ApplescriptResult struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
}
