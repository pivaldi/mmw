//go:generate go-enum --marshal --values
package config

// ENUM(development, staging, production, testing)
type Environment string

func (e Environment) IsDev() bool {
	return e == EnvironmentDevelopment
}
