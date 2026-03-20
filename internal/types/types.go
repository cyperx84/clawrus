package types

// GroupConfig is the top-level config file structure
type GroupConfig struct {
	Groups map[string]Group `yaml:"groups" json:"groups"`
}

// ClawrusConfig is the main config (~/.clawrus/config.yaml)
type ClawrusConfig struct {
	Gateway GatewayConfig `yaml:"gateway" json:"gateway"`
}

// GatewayConfig holds gateway connection settings
type GatewayConfig struct {
	URL string `yaml:"url,omitempty" json:"url,omitempty"`
}

// Group defines a named collection of threads with shared defaults
type Group struct {
	Model    string   `yaml:"model,omitempty" json:"model,omitempty"`
	Thinking string   `yaml:"thinking,omitempty" json:"thinking,omitempty"`
	Timeout  int      `yaml:"timeout,omitempty" json:"timeout,omitempty"` // seconds
	Threads  []Thread `yaml:"threads" json:"threads"`
}

// Thread is a single Discord thread in a group
type Thread struct {
	ID       string `yaml:"id" json:"id"`
	Name     string `yaml:"name,omitempty" json:"name,omitempty"`
	Model    string `yaml:"model,omitempty" json:"model,omitempty"`       // per-thread override
	Thinking string `yaml:"thinking,omitempty" json:"thinking,omitempty"` // per-thread override
	Timeout  int    `yaml:"timeout,omitempty" json:"timeout,omitempty"`   // per-thread override
	Prompt   string `yaml:"prompt,omitempty" json:"prompt,omitempty"`     // per-thread prompt
}

// PresetConfig is the top-level presets file structure
type PresetConfig struct {
	Presets map[string]Preset `yaml:"presets" json:"presets"`
}

// Preset maps a short name to one or more groups
type Preset struct {
	Groups []string `yaml:"groups" json:"groups"`
}

// RunResult captures the outcome of sending to a single thread
type RunResult struct {
	ThreadID   string `json:"thread_id"`
	ThreadName string `json:"thread_name,omitempty"`
	OK         bool   `json:"ok"`
	Error      string `json:"error,omitempty"`
	Reply      string `json:"reply,omitempty"`
}
