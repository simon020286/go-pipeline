package builder

import (
	"embed"
	"fmt"
	"log"
)

//go:embed services/*.yaml
var embeddedServices embed.FS

// globalServiceRegistry is the global service registry
var globalServiceRegistry *ServiceRegistry

func init() {
	// Initialize the global registry
	globalServiceRegistry = NewServiceRegistry()

	// Load embedded services
	if err := globalServiceRegistry.LoadServicesFromEmbed(embeddedServices, "services"); err != nil {
		log.Printf("Warning: failed to load embedded services: %v", err)
	} else {
		log.Printf("Loaded %d embedded service(s): %v", globalServiceRegistry.Count(), globalServiceRegistry.List())
	}

	// Load custom services from user directory
	customServicesPath := GetServicesPath()
	if err := globalServiceRegistry.LoadServicesFromDirectory(customServicesPath); err != nil {
		log.Printf("Warning: failed to load custom services from %s: %v", customServicesPath, err)
	} else if globalServiceRegistry.Count() > 0 {
		log.Printf("Custom services path: %s", customServicesPath)
	}

	// Register all services as step types
	if err := RegisterDynamicAPIServices(globalServiceRegistry); err != nil {
		log.Printf("Warning: failed to register dynamic API services: %v", err)
	} else {
		log.Printf("Registered %d dynamic API step type(s)", globalServiceRegistry.Count())
	}
}

// GetGlobalServiceRegistry returns the global service registry
func GetGlobalServiceRegistry() *ServiceRegistry {
	return globalServiceRegistry
}

// ReloadServices reloads all services (useful for runtime updates)
func ReloadServices() error {
	// Create a new registry
	newRegistry := NewServiceRegistry()

	// Load embedded services
	if err := newRegistry.LoadServicesFromEmbed(embeddedServices, "services"); err != nil {
		return fmt.Errorf("failed to load embedded services: %w", err)
	}

	// Load custom services
	customServicesPath := GetServicesPath()
	if err := newRegistry.LoadServicesFromDirectory(customServicesPath); err != nil {
		return fmt.Errorf("failed to load custom services: %w", err)
	}

	// Register services
	if err := RegisterDynamicAPIServices(newRegistry); err != nil {
		return fmt.Errorf("failed to register services: %w", err)
	}

	// Update the global registry
	globalServiceRegistry = newRegistry

	log.Printf("Reloaded %d service(s): %v", globalServiceRegistry.Count(), globalServiceRegistry.List())
	return nil
}
