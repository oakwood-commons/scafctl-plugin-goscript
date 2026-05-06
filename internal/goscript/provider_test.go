package goscript

import (
	"context"
	"errors"
	"testing"

	sdkprovider "github.com/oakwood-commons/scafctl-plugin-sdk/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testScript = `package main

import "strings"

func Run(input map[string]interface{}) (interface{}, error) {
	name, _ := input["name"].(string)
	return map[string]interface{}{
		"upper": strings.ToUpper(name),
		"length": len(name),
	}, nil
}`

func TestGetProviders(t *testing.T) {
	p := &Plugin{}
	providers, err := p.GetProviders(context.Background())
	require.NoError(t, err)
	assert.Equal(t, []string{ProviderName}, providers)
}

func TestGetProviderDescriptor(t *testing.T) {
	p := &Plugin{}

	t.Run("known provider", func(t *testing.T) {
		desc, err := p.GetProviderDescriptor(context.Background(), ProviderName)
		require.NoError(t, err)
		assert.Equal(t, ProviderName, desc.Name)
		assert.Equal(t, "Go Script Provider", desc.DisplayName)
		assert.NotEmpty(t, desc.Description)
		assert.NotNil(t, desc.Schema)
		require.NotEmpty(t, desc.OutputSchemas)
		assert.Contains(t, desc.Capabilities, sdkprovider.CapabilityFrom)
		assert.Contains(t, desc.Capabilities, sdkprovider.CapabilityTransform)
		assert.Contains(t, desc.Capabilities, sdkprovider.CapabilityAction)
		assert.Contains(t, desc.OutputSchemas, sdkprovider.CapabilityFrom)
		assert.Contains(t, desc.OutputSchemas, sdkprovider.CapabilityTransform)
		assert.Contains(t, desc.OutputSchemas, sdkprovider.CapabilityAction)
	})

	t.Run("unknown provider", func(t *testing.T) {
		_, err := p.GetProviderDescriptor(context.Background(), "unknown")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown provider")
	})
}

func TestGetProviderDescriptor_ValidateDescriptor(t *testing.T) {
	p := &Plugin{}

	desc, err := p.GetProviderDescriptor(context.Background(), ProviderName)
	require.NoError(t, err)
	require.NoError(t, sdkprovider.ValidateDescriptor(desc))
}

func TestExecuteProvider(t *testing.T) {
	p := &Plugin{}

	tests := []struct {
		name    string
		input   map[string]any
		assert  func(*testing.T, any)
		wantErr bool
		errMsg  string
	}{
		{
			name: "basic input",
			input: map[string]any{
				"script": testScript,
				"data":   map[string]any{"name": "hello"},
			},
			assert: func(t *testing.T, data any) {
				result, ok := data.(map[string]any)
				require.True(t, ok, "expected Data to be map[string]any")
				assert.Equal(t, "HELLO", result["upper"])
				assert.Equal(t, 5, result["length"])
			},
		},
		{
			name: "typed string map input",
			input: map[string]any{
				"script": testScript,
				"data":   map[string]string{"name": "world"},
			},
			assert: func(t *testing.T, data any) {
				result, ok := data.(map[string]any)
				require.True(t, ok, "expected Data to be map[string]any")
				assert.Equal(t, "WORLD", result["upper"])
				assert.Equal(t, 5, result["length"])
			},
		},
		{
			name:    "missing script",
			input:   map[string]any{"data": map[string]any{"name": "hello"}},
			wantErr: true,
			errMsg:  "script is required",
		},
		{
			name: "invalid data type",
			input: map[string]any{
				"script": testScript,
				"data":   "hello",
			},
			wantErr: true,
			errMsg:  "data must be a map with string keys",
		},
		{
			name: "missing run function",
			input: map[string]any{
				"script": "package main\nfunc NotRun(input map[string]interface{}) (interface{}, error) { return nil, nil }",
			},
			wantErr: true,
			errMsg:  "load Run function",
		},
		{
			name: "invalid run signature",
			input: map[string]any{
				"script": "package main\nfunc Run() error { return nil }",
			},
			wantErr: true,
			errMsg:  errInvalidRunSignature.Error(),
		},
		{
			name: "runtime error",
			input: map[string]any{
				"script": `package main

import "errors"

func Run(input map[string]interface{}) (interface{}, error) {
	return nil, errors.New("boom")
}`,
			},
			wantErr: true,
			errMsg:  "run script: boom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := p.ExecuteProvider(context.Background(), ProviderName, tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, out)
			tt.assert(t, out.Data)
		})
	}
}

func TestExecuteProvider_UnknownProvider(t *testing.T) {
	p := &Plugin{}
	_, err := p.ExecuteProvider(context.Background(), "unknown", nil)
	assert.Error(t, err)
}

func TestDescribeWhatIf(t *testing.T) {
	p := &Plugin{}

	t.Run("with value", func(t *testing.T) {
		desc, err := p.DescribeWhatIf(context.Background(), ProviderName, map[string]any{
			"script": testScript,
			"data":   map[string]any{"name": "test"},
		})
		require.NoError(t, err)
		assert.Contains(t, desc, "1 input field")
	})

	t.Run("empty value", func(t *testing.T) {
		desc, err := p.DescribeWhatIf(context.Background(), ProviderName, map[string]any{"script": testScript})
		require.NoError(t, err)
		assert.Contains(t, desc, "empty input map")
	})

	t.Run("missing script", func(t *testing.T) {
		_, err := p.DescribeWhatIf(context.Background(), ProviderName, map[string]any{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "script is required")
	})
}

func BenchmarkExecuteProvider(b *testing.B) {
	p := &Plugin{}
	input := map[string]any{
		"script": testScript,
		"data":   map[string]any{"name": "bench"},
	}
	ctx := context.Background()

	b.ReportAllocs()
	b.ResetTimer()

	for b.Loop() {
		_, _ = p.ExecuteProvider(ctx, ProviderName, input)
	}
}

func TestDataFromInput(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		want    map[string]any
		wantErr error
	}{
		{
			name:  "nil input",
			input: nil,
			want:  map[string]any{},
		},
		{
			name:  "missing data",
			input: map[string]any{"script": testScript},
			want:  map[string]any{},
		},
		{
			name:  "string keyed map",
			input: map[string]any{"data": map[string]int{"value": 3}},
			want:  map[string]any{"value": 3},
		},
		{
			name:    "non map",
			input:   map[string]any{"data": 123},
			wantErr: errors.New("data must be a map with string keys"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := dataFromInput(tt.input)
			if tt.wantErr != nil {
				require.Error(t, err)
				assert.EqualError(t, err, tt.wantErr.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
