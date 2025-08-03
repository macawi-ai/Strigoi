module github.com/macawi-ai/strigoi

go 1.23.0

toolchain go1.24.5

require (
	github.com/chzyer/readline v1.5.1
	github.com/creack/pty v1.1.21
	github.com/fatih/color v1.18.0
	github.com/google/uuid v1.6.0
	github.com/marcboeker/go-duckdb v1.8.2
	golang.org/x/time v0.12.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/goccy/go-json v0.10.5 // indirect
	github.com/google/flatbuffers v25.1.24+incompatible // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/klauspost/cpuid/v2 v2.2.9 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/pierrec/lz4/v4 v4.1.22 // indirect
	github.com/rogpeppe/go-internal v1.14.1 // indirect
	github.com/spf13/cobra v1.9.1 // indirect
	github.com/spf13/pflag v1.0.7 // indirect
	github.com/zeebo/xxh3 v1.0.2 // indirect
	golang.org/x/exp v0.0.0-20250128182459-e0ece0dbea4c // indirect
	golang.org/x/mod v0.22.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/tools v0.29.0 // indirect
	golang.org/x/xerrors v0.0.0-20240903120638-7835f813f4da // indirect
	google.golang.org/protobuf v1.31.0 // indirect
)

replace github.com/macawi-ai/strigoi/modules/mcp/config => ./modules.bak/mcp/config

replace github.com/macawi-ai/strigoi/modules/mcp/privilege => ./modules.bak/mcp/privilege

replace github.com/macawi-ai/strigoi/modules/mcp/session => ./modules.bak/mcp/session

replace github.com/macawi-ai/strigoi/modules/mcp/stdio => ./modules.bak/mcp/stdio

replace github.com/macawi-ai/strigoi/modules/mcp/validation => ./modules.bak/mcp/validation

replace github.com/macawi-ai/strigoi/internal/modules/mcp => ./internal/packages

replace github.com/macawi/strigoi/internal/actors => ./internal/actors
