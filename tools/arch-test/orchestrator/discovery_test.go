package orchestrator

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiscoverServices(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()
	servicesDir := filepath.Join(tmpDir, "services")

	// Create mock services
	service1 := filepath.Join(servicesDir, "service1")
	service2 := filepath.Join(servicesDir, "service2")
	notService := filepath.Join(servicesDir, "README.md")

	os.MkdirAll(service1, 0755)
	os.MkdirAll(service2, 0755)
	os.WriteFile(notService, []byte("test"), 0644)

	// Create mise.toml for service1
	os.WriteFile(filepath.Join(service1, "mise.toml"), []byte(`
[tasks."arch:check"]
run = "arch-go check"
`), 0644)

	services, err := DiscoverServices(servicesDir)
	if err != nil {
		t.Fatalf("DiscoverServices failed: %v", err)
	}

	if len(services) != 1 {
		t.Errorf("Expected 1 service, got %d", len(services))
	}

	if services[0].Name != "service1" {
		t.Errorf("Expected service name 'service1', got '%s'", services[0].Name)
	}

	if !services[0].HasArchCheck {
		t.Errorf("Expected service1 to have arch:check task")
	}
}

func TestDiscoverServices_NoServicesDir(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "nonexistent")

	services, err := DiscoverServices(nonExistent)
	if err == nil {
		t.Error("Expected error for non-existent directory")
	}
	if services != nil {
		t.Error("Expected nil services on error")
	}
}
