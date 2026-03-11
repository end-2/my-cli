package config

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"text/template"

	"github.com/spf13/viper"
)

const (
	DefaultFileName = "config.yaml"
	defaultEtcDir   = "/etc"
)

var ErrInvalidTarget = errors.New("config target must be a non-nil pointer")
var ErrConfigNotFound = errors.New("no config file found")

type Loader struct {
	appName        string
	file           string
	explicitPath   string
	currentDir     func() (string, error)
	homeDir        func() (string, error)
	executablePath func() (string, error)
	etcDir         string
}

type configCandidate struct {
	path     string
	required bool
}

func New(file string) *Loader {
	if file == "" {
		file = DefaultFileName
	}

	return &Loader{
		file:           filepath.Base(file),
		explicitPath:   file,
		currentDir:     os.Getwd,
		homeDir:        os.UserHomeDir,
		executablePath: os.Executable,
		etcDir:         defaultEtcDir,
	}
}

func NewForApp(appName, file string) *Loader {
	if file == "" {
		file = DefaultFileName
	}

	return &Loader{
		appName:        appName,
		file:           file,
		currentDir:     os.Getwd,
		homeDir:        os.UserHomeDir,
		executablePath: os.Executable,
		etcDir:         defaultEtcDir,
	}
}

func Load(target any) error {
	return NewForApp("", DefaultFileName).Load(target)
}

func LoadForApp(appName string, target any) error {
	return NewForApp(appName, DefaultFileName).Load(target)
}

func LoadFile(file string, target any) error {
	return New(file).Load(target)
}

func IsConfigNotFound(err error) bool {
	return errors.Is(err, ErrConfigNotFound)
}

func (l *Loader) Load(target any) error {
	if err := validateTarget(target); err != nil {
		return err
	}

	return l.loadMerged(target)
}

func (l *Loader) loadMerged(target any) error {
	appName, err := l.resolveAppName()
	if err != nil {
		return err
	}

	candidates, err := l.configCandidates(appName)
	if err != nil {
		return err
	}

	v := viper.New()
	loadedPaths := make([]string, 0, len(candidates))

	for _, candidate := range candidates {
		if !candidate.required {
			exists, err := fileExists(candidate.path)
			if err != nil {
				return fmt.Errorf("stat config file %q: %w", candidate.path, err)
			}
			if !exists {
				continue
			}
		}

		cfg, err := readConfigFile(candidate.path)
		if err != nil {
			return err
		}

		if err := v.MergeConfigMap(cfg.AllSettings()); err != nil {
			return fmt.Errorf("merge config file %q: %w", candidate.path, err)
		}

		loadedPaths = append(loadedPaths, candidate.path)
	}

	if len(loadedPaths) == 0 {
		return fmt.Errorf("%w in any of %s", ErrConfigNotFound, strings.Join(candidatePaths(candidates), ", "))
	}

	if err := v.Unmarshal(target); err != nil {
		return fmt.Errorf("decode merged config: %w", err)
	}

	return nil
}

func (l *Loader) resolveAppName() (string, error) {
	if l.appName != "" {
		return l.appName, nil
	}

	executablePath, err := l.executablePath()
	if err != nil {
		return "", fmt.Errorf("resolve app name from executable: %w", err)
	}

	appName := filepath.Base(executablePath)
	if appName == "" || appName == "." || appName == string(filepath.Separator) {
		return "", errors.New("resolve app name from executable: empty executable name")
	}

	return appName, nil
}

func (l *Loader) configCandidates(appName string) ([]configCandidate, error) {
	searchPaths, err := l.searchPaths(appName)
	if err != nil {
		return nil, err
	}

	candidates := make([]configCandidate, 0, len(searchPaths)+1)
	for _, path := range searchPaths {
		candidates = append(candidates, configCandidate{path: path})
	}

	if l.explicitPath != "" {
		candidates = append(candidates, configCandidate{
			path:     l.explicitPath,
			required: true,
		})
	}

	return candidates, nil
}

func (l *Loader) searchPaths(appName string) ([]string, error) {
	currentDir, err := l.currentDir()
	if err != nil {
		return nil, fmt.Errorf("resolve current directory: %w", err)
	}

	homeDir, err := l.homeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}

	return []string{
		filepath.Join(l.etcDir, appName, l.file),
		filepath.Join(homeDir, l.file),
		filepath.Join(currentDir, l.file),
	}, nil
}

func validateTarget(target any) error {
	value := reflect.ValueOf(target)
	if !value.IsValid() || value.Kind() != reflect.Pointer || value.IsNil() {
		return ErrInvalidTarget
	}

	return nil
}

func readConfigFile(file string) (*viper.Viper, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read config file %q: %w", file, err)
	}

	rendered, err := renderConfigTemplate(file, content)
	if err != nil {
		return nil, err
	}

	v := viper.New()

	if ext := strings.TrimPrefix(filepath.Ext(file), "."); ext != "" {
		v.SetConfigType(ext)
	}

	if err := v.ReadConfig(bytes.NewReader(rendered)); err != nil {
		return nil, fmt.Errorf("read config file %q: %w", file, err)
	}

	return v, nil
}

func renderConfigTemplate(file string, content []byte) ([]byte, error) {
	tmpl, err := template.New(filepath.Base(file)).
		Option("missingkey=error").
		Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse config template %q: %w", file, err)
	}

	var rendered bytes.Buffer
	if err := tmpl.Execute(&rendered, environmentMap()); err != nil {
		return nil, fmt.Errorf("render config template %q: %w", file, err)
	}

	return rendered.Bytes(), nil
}

func candidatePaths(candidates []configCandidate) []string {
	paths := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		paths = append(paths, candidate.path)
	}

	return paths
}

func environmentMap() map[string]string {
	env := make(map[string]string)

	for _, entry := range os.Environ() {
		key, value, found := strings.Cut(entry, "=")
		if !found {
			continue
		}

		env[key] = value
	}

	return env
}

func fileExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}

	if os.IsNotExist(err) {
		return false, nil
	}

	return false, err
}
