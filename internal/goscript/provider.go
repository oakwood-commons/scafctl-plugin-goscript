// Package goscript implements the Go script provider plugin.
package goscript

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"github.com/Masterminds/semver/v3"
	"github.com/google/jsonschema-go/jsonschema"
	sdkplugin "github.com/oakwood-commons/scafctl-plugin-sdk/plugin"
	sdkprovider "github.com/oakwood-commons/scafctl-plugin-sdk/provider"
	sdkhelper "github.com/oakwood-commons/scafctl-plugin-sdk/provider/schemahelper"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

const (
	// ProviderName is the unique identifier for this provider.
	ProviderName = "goscript"

	// Version is the provider version.
	Version = "0.1.0"
)

var errInvalidRunSignature = errors.New("script must define func Run(input map[string]interface{}) (interface{}, error)")

// Plugin implements the scafctl ProviderPlugin interface.
type Plugin struct{}

// GetProviders returns the list of providers exposed by this plugin.
//
//nolint:revive // ctx required by interface
func (p *Plugin) GetProviders(_ context.Context) ([]string, error) {
	return []string{ProviderName}, nil
}

// GetProviderDescriptor returns the descriptor for the named provider.
//
//nolint:revive // ctx required by interface
func (p *Plugin) GetProviderDescriptor(_ context.Context, providerName string) (*sdkprovider.Descriptor, error) {
	if providerName != ProviderName {
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}

	return &sdkprovider.Descriptor{
		Name:        ProviderName,
		DisplayName: "Go Script Provider",
		Description: "Executes inline Go scripts with a yaegi interpreter",
		APIVersion:  "v1",
		Version:     semver.MustParse(Version),
		Category:    "script",
		Capabilities: []sdkprovider.Capability{
			sdkprovider.CapabilityFrom,
			sdkprovider.CapabilityTransform,
			sdkprovider.CapabilityAction,
		},
		Schema: sdkhelper.ObjectSchema(
			[]string{"script"},
			map[string]*jsonschema.Schema{
				"script": sdkhelper.StringProp(
					"Go source code that defines func Run(input map[string]interface{}) (interface{}, error)",
					sdkhelper.WithExample("package main\n\nfunc Run(input map[string]interface{}) (interface{}, error) {\n\treturn input, nil\n}"),
				),
				"data": {
					Description: "Optional map input passed to Run as its input parameter",
				},
			},
		),
		OutputSchemas: map[sdkprovider.Capability]*jsonschema.Schema{
			sdkprovider.CapabilityFrom:      anyOutputSchema("Provider result returned by Run when used in resolve/from mode"),
			sdkprovider.CapabilityTransform: anyOutputSchema("Provider result returned by Run when used in transform mode"),
			sdkprovider.CapabilityAction:    actionOutputSchema(),
		},
	}, nil
}

// ExecuteProvider executes the named provider with the given input.
//
//nolint:revive // ctx required by interface
func (p *Plugin) ExecuteProvider(_ context.Context, providerName string, input map[string]any) (*sdkprovider.Output, error) {
	if providerName != ProviderName {
		return nil, fmt.Errorf("unknown provider: %s", providerName)
	}

	script, err := scriptFromInput(input)
	if err != nil {
		return nil, err
	}

	data, err := dataFromInput(input)
	if err != nil {
		return nil, err
	}

	result, err := executeScript(script, data)
	if err != nil {
		return nil, fmt.Errorf("execute go script: %w", err)
	}

	return &sdkprovider.Output{
		Data: result,
	}, nil
}

// DescribeWhatIf returns a description of what the provider would do.
//
//nolint:revive // ctx required by interface
func (p *Plugin) DescribeWhatIf(_ context.Context, providerName string, input map[string]any) (string, error) {
	if providerName != ProviderName {
		return "", fmt.Errorf("unknown provider: %s", providerName)
	}

	_, err := scriptFromInput(input)
	if err != nil {
		return "", err
	}

	data, err := dataFromInput(input)
	if err != nil {
		return "", err
	}

	if len(data) == 0 {
		return "Would execute the Go script with an empty input map", nil
	}

	return fmt.Sprintf("Would execute the Go script with %d input field(s)", len(data)), nil
}

// ConfigureProvider stores host-side configuration.
//
//nolint:revive // ctx and cfg required by interface
func (p *Plugin) ConfigureProvider(_ context.Context, _ string, _ sdkplugin.ProviderConfig) error {
	return nil
}

// ExecuteProviderStream is not supported.
//
//nolint:revive // all params required by interface
func (p *Plugin) ExecuteProviderStream(_ context.Context, _ string, _ map[string]any, _ func(sdkplugin.StreamChunk)) error {
	return sdkplugin.ErrStreamingNotSupported
}

// ExtractDependencies returns resolver keys this input depends on.
//
//nolint:revive // all params required by interface
func (p *Plugin) ExtractDependencies(_ context.Context, _ string, _ map[string]any) ([]string, error) {
	return nil, nil
}

// StopProvider performs cleanup for the named provider.
//
//nolint:revive // all params required by interface
func (p *Plugin) StopProvider(_ context.Context, _ string) error {
	return nil
}

func scriptFromInput(input map[string]any) (string, error) {
	if input == nil {
		return "", errors.New("script is required")
	}

	script, ok := input["script"].(string)
	if !ok || script == "" {
		return "", errors.New("script is required")
	}

	return script, nil
}

func dataFromInput(input map[string]any) (map[string]any, error) {
	if input == nil {
		return map[string]any{}, nil
	}

	data, exists := input["data"]
	if !exists || data == nil {
		return map[string]any{}, nil
	}

	if typed, ok := data.(map[string]any); ok {
		return typed, nil
	}

	value := reflect.ValueOf(data)
	if value.Kind() != reflect.Map {
		return nil, errors.New("data must be a map with string keys")
	}

	if value.Type().Key().Kind() != reflect.String {
		return nil, errors.New("data must be a map with string keys")
	}

	normalized := make(map[string]any, value.Len())
	iter := value.MapRange()
	for iter.Next() {
		normalized[iter.Key().String()] = iter.Value().Interface()
	}

	return normalized, nil
}

func executeScript(script string, input map[string]any) (any, error) {
	interpreter := interp.New(interp.Options{})
	if err := interpreter.Use(stdlib.Symbols); err != nil {
		return nil, fmt.Errorf("load stdlib symbols: %w", err)
	}

	if _, err := interpreter.Eval(script); err != nil {
		return nil, fmt.Errorf("evaluate script: %w", err)
	}

	runValue, err := interpreter.Eval("Run")
	if err != nil {
		return nil, fmt.Errorf("load Run function: %w", err)
	}

	run, ok := runValue.Interface().(func(map[string]interface{}) (interface{}, error))
	if !ok {
		return nil, errInvalidRunSignature
	}

	result, err := run(input)
	if err != nil {
		return nil, fmt.Errorf("run script: %w", err)
	}

	return result, nil
}

func anyOutputSchema(description string) *jsonschema.Schema {
	return &jsonschema.Schema{
		Description: description,
	}
}

func actionOutputSchema() *jsonschema.Schema {
	return sdkhelper.ObjectSchema(
		[]string{"success"},
		map[string]*jsonschema.Schema{
			"success": sdkhelper.BoolProp(
				"Whether the Go script action completed successfully",
			),
			"data": {
				Description: "Provider result returned by Run when used in action mode",
			},
		},
	)
}
