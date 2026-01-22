package registry

import (
	"fmt"

	infra "chunisupport-song-batch/internal/infra/datasource"
)

// Definition はデータソースをダウンロードするために必要なメタデータを保持します。
type Definition struct {
	Type   string
	URL    string
	Params any
}

// Provider はデータソースの定義を構築します。
type Provider func() (Definition, error)

// providers はデータソース名とそのプロバイダーを関連付けます。
var providers = map[string]Provider{}

// Register はデータソース名とそのプロバイダーを関連付けます。
func Register(name string, provider Provider) {
	if provider == nil {
		panic("registry: provider must not be nil")
	}
	if _, exists := providers[name]; exists {
		panic(fmt.Sprintf("registry: provider for %s already registered", name))
	}
	providers[name] = provider
}

// Resolve は指定された名前のデータソース定義を構築します。
func Resolve(name string, active bool) (infra.Datasource, error) {
	provider, ok := providers[name]
	if !ok {
		return infra.Datasource{}, fmt.Errorf("datasource provider %q not found", name)
	}

	definition, err := provider()
	if err != nil {
		return infra.Datasource{}, fmt.Errorf("failed to build datasource %q: %w", name, err)
	}

	if definition.Type == "" {
		definition.Type = name
	}

	return infra.Datasource{
		Type:   definition.Type,
		URL:    definition.URL,
		Params: definition.Params,
		Active: active,
	}, nil
}
