package marketplace

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// Logger is a simple logger interface to avoid circular imports
type Logger interface {
	Info(format string, args ...interface{})
	Warn(format string, args ...interface{})
	Error(format string, args ...interface{})
	Success(format string, args ...interface{})
}

// Client handles marketplace operations
type Client struct {
	baseURL       string
	cacheDir      string
	modulesDir    string
	httpClient    *http.Client
	verifier      *SHA256Verifier
	trustManager  *TrustManager
	logger        Logger
}

// NewClient creates a new marketplace client
func NewClient(cacheDir, modulesDir string, logger Logger) *Client {
	return &Client{
		baseURL:    "https://raw.githubusercontent.com/macawi-ai/marketplace/main",
		cacheDir:   cacheDir,
		modulesDir: modulesDir,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		verifier:     NewSHA256Verifier(),
		trustManager: NewTrustManager(),
		logger:       logger,
	}
}

// Search searches for modules in the marketplace
func (c *Client) Search(query string) ([]ModuleManifest, error) {
	// For now, implement a simple search that fetches catalog
	// In production, this would query a proper index
	
	manifests := []ModuleManifest{}
	
	// Search official modules
	officialManifests, err := c.searchInNamespace("official", query)
	if err != nil {
		c.logger.Warn("Failed to search official modules: %v", err)
	} else {
		manifests = append(manifests, officialManifests...)
	}
	
	// Search community modules
	communityManifests, err := c.searchInNamespace("community", query)
	if err != nil {
		c.logger.Warn("Failed to search community modules: %v", err)
	} else {
		manifests = append(manifests, communityManifests...)
	}
	
	return manifests, nil
}

// InstallModule downloads and installs a module
func (c *Client) InstallModule(moduleID string, version string) error {
	c.logger.Info("Installing module: %s version %s", moduleID, version)
	
	// Parse module path (e.g., "mcp/sudo-tailgate" or "johnsmith/custom-scanner")
	namespace, modulePath := c.parseModuleID(moduleID)
	
	// Fetch manifest
	manifest, err := c.fetchManifest(namespace, modulePath, version)
	if err != nil {
		return fmt.Errorf("failed to fetch manifest: %w", err)
	}
	
	// Check trust level and prompt if needed
	if !manifest.IsOfficial() {
		if !c.trustManager.PromptThirdPartyWarning(manifest) {
			return fmt.Errorf("installation cancelled by user")
		}
	}
	
	// Download module package
	c.logger.Info("Downloading module from %s", manifest.StrigoiModule.Distribution.URI)
	data, err := c.download(manifest.StrigoiModule.Distribution.URI)
	if err != nil {
		return fmt.Errorf("failed to download module: %w", err)
	}
	
	// Verify SHA-256
	c.logger.Info("Verifying module integrity...")
	if !c.verifier.Verify(data, manifest.StrigoiModule.Distribution.Verification.Hash) {
		actualHash := c.verifier.ComputeHash(data)
		return IntegrityError{
			Module:       moduleID,
			ExpectedHash: manifest.StrigoiModule.Distribution.Verification.Hash,
			ActualHash:   actualHash,
		}
	}
	
	// Verify size
	if int64(len(data)) != manifest.StrigoiModule.Distribution.Verification.SizeBytes {
		return fmt.Errorf("size mismatch: expected %d bytes, got %d bytes",
			manifest.StrigoiModule.Distribution.Verification.SizeBytes, len(data))
	}
	
	// Install to local modules directory
	installPath := filepath.Join(c.modulesDir, namespace, modulePath, version)
	if err := c.install(manifest, data, installPath); err != nil {
		return fmt.Errorf("failed to install module: %w", err)
	}
	
	c.logger.Success("Module installed successfully: %s", installPath)
	return nil
}

// UpdateCache updates the local marketplace cache
func (c *Client) UpdateCache() error {
	c.logger.Info("Updating marketplace cache...")
	
	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(c.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}
	
	// For now, just create a timestamp file
	// In production, this would sync the marketplace repository
	timestampFile := filepath.Join(c.cacheDir, "last_update")
	if err := ioutil.WriteFile(timestampFile, []byte(time.Now().Format(time.RFC3339)), 0644); err != nil {
		return fmt.Errorf("failed to write timestamp: %w", err)
	}
	
	c.logger.Success("Marketplace cache updated")
	return nil
}

// ListInstalled lists all installed modules
func (c *Client) ListInstalled() ([]InstalledModule, error) {
	modules := []InstalledModule{}
	
	// Walk through modules directory
	err := filepath.Walk(c.modulesDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Look for manifest.yaml files
		if info.Name() == "manifest.yaml" {
			manifest, err := c.loadManifestFromFile(path)
			if err != nil {
				c.logger.Warn("Failed to load manifest from %s: %v", path, err)
				return nil
			}
			
			relPath, _ := filepath.Rel(c.modulesDir, filepath.Dir(path))
			modules = append(modules, InstalledModule{
				Path:     relPath,
				Manifest: manifest,
			})
		}
		
		return nil
	})
	
	return modules, err
}

// Helper methods

func (c *Client) parseModuleID(moduleID string) (namespace string, modulePath string) {
	parts := strings.SplitN(moduleID, "/", 2)
	if len(parts) == 1 {
		// No namespace specified, assume official
		return "official", moduleID
	}
	
	// Check if first part looks like a username (for community modules)
	if !strings.Contains(parts[0], "-") && !strings.Contains(parts[0], "_") {
		// Likely a username, so this is a community module
		return "community/" + parts[0], parts[1]
	}
	
	// Otherwise, it's an official module with category
	return "official", moduleID
}

func (c *Client) fetchManifest(namespace, modulePath, version string) (*ModuleManifest, error) {
	// Construct manifest URL
	url := fmt.Sprintf("%s/modules/%s/%s/v%s.yaml", c.baseURL, namespace, modulePath, version)
	
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manifest not found: %s", resp.Status)
	}
	
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}
	
	var manifest ModuleManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}
	
	return &manifest, nil
}

func (c *Client) download(uri string) ([]byte, error) {
	resp, err := c.httpClient.Get(uri)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download failed: %s", resp.Status)
	}
	
	return ioutil.ReadAll(resp.Body)
}

func (c *Client) install(manifest *ModuleManifest, data []byte, installPath string) error {
	// Create installation directory
	if err := os.MkdirAll(installPath, 0755); err != nil {
		return fmt.Errorf("failed to create install directory: %w", err)
	}
	
	// Write module package
	packagePath := filepath.Join(installPath, "module.tar.gz")
	if err := ioutil.WriteFile(packagePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write module package: %w", err)
	}
	
	// Write manifest for reference
	manifestPath := filepath.Join(installPath, "manifest.yaml")
	manifestData, err := yaml.Marshal(manifest)
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}
	
	if err := ioutil.WriteFile(manifestPath, manifestData, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}
	
	// TODO: Extract tar.gz and set up module
	
	return nil
}

func (c *Client) searchInNamespace(namespace, query string) ([]ModuleManifest, error) {
	// Fetch catalog
	catalogURL := fmt.Sprintf("%s/catalog.yaml", c.baseURL)
	resp, err := c.httpClient.Get(catalogURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch catalog: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// For demo purposes, return empty results if catalog not found
		c.logger.Warn("Catalog not found, returning empty results")
		return []ModuleManifest{}, nil
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read catalog: %w", err)
	}

	// Parse catalog
	var catalog struct {
		Marketplace struct {
			Version string    `yaml:"version"`
			Updated time.Time `yaml:"updated"`
		} `yaml:"marketplace"`
		Modules map[string][]struct {
			ID          string `yaml:"id"`
			Name        string `yaml:"name"`
			LatestVersion string `yaml:"latest_version"`
			Description string `yaml:"description"`
			RiskLevel   string `yaml:"risk_level"`
			ManifestURL string `yaml:"manifest_url"`
		} `yaml:"modules"`
	}

	if err := yaml.Unmarshal(data, &catalog); err != nil {
		return nil, fmt.Errorf("failed to parse catalog: %w", err)
	}

	// Search in the appropriate namespace
	manifests := []ModuleManifest{}
	modules, ok := catalog.Modules[namespace]
	if !ok {
		return manifests, nil
	}

	// Simple substring search
	queryLower := strings.ToLower(query)
	for _, mod := range modules {
		if strings.Contains(strings.ToLower(mod.ID), queryLower) ||
		   strings.Contains(strings.ToLower(mod.Name), queryLower) ||
		   strings.Contains(strings.ToLower(mod.Description), queryLower) {
			// Fetch the actual manifest
			manifest, err := c.fetchManifestFromURL(mod.ManifestURL)
			if err != nil {
				c.logger.Warn("Failed to fetch manifest for %s: %v", mod.ID, err)
				continue
			}
			manifests = append(manifests, *manifest)
		}
	}

	return manifests, nil
}

func (c *Client) fetchManifestFromURL(url string) (*ModuleManifest, error) {
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("manifest not found: %s", resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest ModuleManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	return &manifest, nil
}

func (c *Client) loadManifestFromFile(path string) (*ModuleManifest, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	
	var manifest ModuleManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, err
	}
	
	return &manifest, nil
}

// InstalledModule represents an installed module
type InstalledModule struct {
	Path     string
	Manifest *ModuleManifest
}