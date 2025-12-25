package service

type Step struct {
	Step           string `yaml:"step"`
	Script         string `yaml:"script"`
	TimeoutSeconds int64  `yaml:"timeout_seconds"`
	Parallel       bool   `yaml:"parallel"`
}

type Stage struct {
	Stage     string `yaml:"stage"`
	Steps     []Step `yaml:"steps"`
	Artifacts string `yaml:"artifacts"`
}

type PipelineScript struct {
	Stages    []Stage `yaml:"stages"`
	Artifacts string  `yaml:"artifacts"`
}
