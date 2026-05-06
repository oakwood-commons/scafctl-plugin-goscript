# scafctl-plugin-goscript

Yaegi-backed Go script provider for scafctl.

## Installation

```bash
# Build from source
task build

# Or download from releases
gh release download --repo github.com/oakwood-commons/scafctl-plugin-goscript
```

## Usage

Register this plugin in your scafctl configuration, then reference
the `goscript` provider in your solutions. It can run as a `resolve`, `transform`,
or `action` provider, similar to `exec`.

```yaml
resolvers:
  processData:
    resolve:
      with:
        - provider: goscript
          inputs:
            script: |
              package main

              import "strings"

              func Run(input map[string]interface{}) (interface{}, error) {
                  name, _ := input["name"].(string)
                  return map[string]interface{}{
                      "upper": strings.ToUpper(name),
                      "length": len(name),
                  }, nil
              }
            data:
              name: hello
```

The script must define `func Run(input map[string]interface{}) (interface{}, error)`.
`script` is required and `data` is optional.

## Sample Solution

A runnable example lives in `./examples/greeting/solution.yaml`. It feeds static input into the `goscript` provider during `resolve` and prints the result with the built-in `message` action.

```bash
scafctl plugins install -f ./examples/greeting/solution.yaml
scafctl run solution -f ./examples/greeting/solution.yaml
```

Expected output is a small object with a greeting, uppercased language, and uppercased topic list.

## Development

```bash
task test

task lint
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

Apache-2.0 -- see [LICENSE](LICENSE) for details.
