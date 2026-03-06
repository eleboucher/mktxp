package config

import (
	"embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"gopkg.in/yaml.v3"
)

func (h *ConfigHandler) ApplyEnvOverrides() error {
	return NewEnvConfigurator().ApplyRouterOverrides(h)
}

func (h *ConfigHandler) ApplySystemEnvOverrides() error {
	return NewEnvConfigurator().ApplySystemOverrides(h)
}

//go:embed templates/mktxp.yaml templates/_mktxp.yaml
var templateFS embed.FS

var Handler = &ConfigHandler{}

// GetTemplateFS returns the embedded template filesystem for testing.
func GetTemplateFS() embed.FS {
	return templateFS
}

type ConfigHandler struct {
	mu         sync.RWMutex
	cfgDir     string
	mainConfig *mainConfigFile // parsed mktxp.yaml
	sysConfig  *SystemConfig   // parsed _mktxp.yaml
	entryCache map[string]*RouterConfigEntry
}

// mainConfigFile is the top-level structure of mktxp.yaml.
type mainConfigFile struct {
	Default rawEntry            `yaml:"default"`
	Routers map[string]rawEntry `yaml:"routers"`
}

// Init initializes the ConfigHandler with the given config directory.
// If cfgDir is empty, it defaults to ~/mktxp/.
// It creates config files from embedded templates if they do not exist.
func (h *ConfigHandler) Init(cfgDir string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if cfgDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("config: cannot determine home dir: %w", err)
		}
		cfgDir = filepath.Join(home, "mktxp")
	}
	h.cfgDir = cfgDir

	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		return fmt.Errorf("config: create dir %s: %w", cfgDir, err)
	}

	mainPath := filepath.Join(cfgDir, "mktxp.yaml")
	sysPath := filepath.Join(cfgDir, "_mktxp.yaml")

	if err := h.ensureFile(mainPath, "templates/mktxp.yaml"); err != nil {
		return err
	}
	if err := h.ensureFile(sysPath, "templates/_mktxp.yaml"); err != nil {
		return err
	}

	if err := h.loadMain(mainPath); err != nil {
		return err
	}
	if err := h.loadSystem(sysPath); err != nil {
		return err
	}

	h.buildEntryCache()
	return nil
}

// ensureFile copies an embedded template to dst if dst does not exist.
func (h *ConfigHandler) ensureFile(dst, embedPath string) error {
	if _, err := os.Stat(dst); err == nil {
		return nil
	}
	data, err := templateFS.ReadFile(embedPath)
	if err != nil {
		return fmt.Errorf("config: read template %s: %w", embedPath, err)
	}
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		return fmt.Errorf("config: write %s: %w", dst, err)
	}
	slog.Info("Created config file from template", "path", dst)
	return nil
}

func (h *ConfigHandler) loadMain(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("config: read %s: %w", path, err)
	}
	var mc mainConfigFile
	if err := yaml.Unmarshal(data, &mc); err != nil {
		return fmt.Errorf("config: parse %s: %w", path, err)
	}
	h.mainConfig = &mc
	return nil
}

type sysConfigFile struct {
	MKTXP SystemConfig `yaml:"mktxp"`
}

func (h *ConfigHandler) loadSystem(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("config: read %s: %w", path, err)
	}
	var sf sysConfigFile
	if err := yaml.Unmarshal(data, &sf); err != nil {
		return fmt.Errorf("config: parse %s: %w", path, err)
	}
	sc := applySystemDefaults(sf.MKTXP)
	h.sysConfig = &sc
	return nil
}

// buildEntryCache merges per-router entries with defaults and caches them.
func (h *ConfigHandler) buildEntryCache() {
	h.entryCache = make(map[string]*RouterConfigEntry, len(h.mainConfig.Routers))
	defaults := mergeWithDefaults(h.mainConfig.Default)
	for name, raw := range h.mainConfig.Routers {
		merged := mergeEntry(defaults, raw)
		e := merged
		h.entryCache[name] = &e
	}
}

func (h *ConfigHandler) SystemEntry() *SystemConfig {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.sysConfig
}

func (h *ConfigHandler) RegisterTestSystemConfig(cfg *SystemConfig) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sysConfig = cfg
}

// RouterEntry returns the merged config entry for the named router, or nil if not found.
func (h *ConfigHandler) RouterEntry(name string) *RouterConfigEntry {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.entryCache[name]
}

func (h *ConfigHandler) RegisteredEntries() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	names := make([]string, 0, len(h.mainConfig.Routers))
	for name := range h.mainConfig.Routers {
		names = append(names, name)
	}
	return names
}

func (h *ConfigHandler) MainConfPath() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return filepath.Join(h.cfgDir, "mktxp.yaml")
}

func (h *ConfigHandler) SysConfPath() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return filepath.Join(h.cfgDir, "_mktxp.yaml")
}

func (h *ConfigHandler) ConfigDir() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.cfgDir
}

func (h *ConfigHandler) RegisterTestRouterEntry(name string, cfg *RouterConfigEntry) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.mainConfig == nil {
		h.mainConfig = &mainConfigFile{Routers: make(map[string]rawEntry)}
	}
	if h.entryCache == nil {
		h.entryCache = make(map[string]*RouterConfigEntry)
	}
	h.entryCache[name] = cfg
	h.mainConfig.Routers[name] = rawEntry{
		Hostname: &cfg.Hostname,
		Port:     &cfg.Port,
		Username: &cfg.Username,
		Password: &cfg.Password,
		Enabled:  &cfg.Enabled,
		UseSSL:   &cfg.UseSSL,
	}
}

func (h *ConfigHandler) Reload() error {
	h.mu.Lock()
	defer h.mu.Unlock()
	mainPath := filepath.Join(h.cfgDir, "mktxp.yaml")
	sysPath := filepath.Join(h.cfgDir, "_mktxp.yaml")
	if err := h.loadMain(mainPath); err != nil {
		return err
	}
	if err := h.loadSystem(sysPath); err != nil {
		return err
	}
	h.buildEntryCache()
	return nil
}
