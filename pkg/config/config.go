package config

// Validator is a config object that can be validated.
type Validator interface {
	// Validate returns error if config is invalid.
	// Configs must be validated before passed to object constructors.
	Validate() error
}

type Config struct{}
