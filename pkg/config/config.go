package config

// Validatable is a config object that can be validated.
type Validatable interface {
	// Validate returns error if config is invalid.
	// Configs must be validated before passed to object constructors.
	Validate() error
}
