package builder

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/simon020286/go-pipeline/config"
	"gopkg.in/yaml.v3"
)

// ServiceRegistry maintains all loaded service definitions
type ServiceRegistry struct {
	services map[string]*config.ServiceDefinition
}

// NewServiceRegistry creates a new registry
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string]*config.ServiceDefinition),
	}
}

// Register registers a service definition
func (sr *ServiceRegistry) Register(def *config.ServiceDefinition) error {
	if err := def.Validate(); err != nil {
		return fmt.Errorf("invalid service definition: %w", err)
	}
	sr.services[def.Service.Name] = def
	return nil
}

// Get returns a service definition by name
func (sr *ServiceRegistry) Get(name string) (*config.ServiceDefinition, bool) {
	def, exists := sr.services[name]
	return def, exists
}

// List returns all registered service names
func (sr *ServiceRegistry) List() []string {
	names := make([]string, 0, len(sr.services))
	for name := range sr.services {
		names = append(names, name)
	}
	return names
}

// Count returns the number of registered services
func (sr *ServiceRegistry) Count() int {
	return len(sr.services)
}

// LoadServicesFromEmbed loads services from an embed.FS
func (sr *ServiceRegistry) LoadServicesFromEmbed(embedFS embed.FS, basePath string) error {
	entries, err := fs.ReadDir(embedFS, basePath)
	if err != nil {
		return fmt.Errorf("failed to read embedded services directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Load only .yaml or .yml files
		if !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		filePath := filepath.Join(basePath, entry.Name())
		data, err := fs.ReadFile(embedFS, filePath)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", filePath, err)
		}

		if err := sr.loadServiceFromBytes(data, entry.Name()); err != nil {
			return fmt.Errorf("failed to load embedded service %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// LoadServicesFromDirectory loads services from a filesystem directory
func (sr *ServiceRegistry) LoadServicesFromDirectory(dirPath string) error {
	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		// Directory doesn't exist, not an error - simply no custom services
		return nil
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read services directory %s: %w", dirPath, err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		// Load only .yaml or .yml files
		if !strings.HasSuffix(entry.Name(), ".yaml") && !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		filePath := filepath.Join(dirPath, entry.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", filePath, err)
		}

		if err := sr.loadServiceFromBytes(data, entry.Name()); err != nil {
			// Log warning but continue with other services
			fmt.Printf("Warning: failed to load service from %s: %v\n", entry.Name(), err)
			continue
		}
	}

	return nil
}

// loadServiceFromBytes loads a service definition from bytes
func (sr *ServiceRegistry) loadServiceFromBytes(data []byte, filename string) error {
	var def config.ServiceDefinition
	if err := yaml.Unmarshal(data, &def); err != nil {
		return fmt.Errorf("failed to parse YAML: %w", err)
	}

	// If name is not specified, use the filename
	if def.Service.Name == "" {
		def.Service.Name = strings.TrimSuffix(filename, filepath.Ext(filename))
	}

	if err := sr.Register(&def); err != nil {
		return err
	}

	return nil
}

// GetServicesPath returns the path to the custom services directory
// Checks environment variable first, then uses default directory
func GetServicesPath() string {
	// Check environment variable
	if path := os.Getenv("GO_PIPELINE_SERVICES_PATH"); path != "" {
		return path
	}

	// Default: ~/.go-pipeline/services
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./services" // fallback to local directory
	}

	return filepath.Join(homeDir, ".go-pipeline", "services")
}
