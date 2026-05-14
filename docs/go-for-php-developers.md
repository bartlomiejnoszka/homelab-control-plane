# Go Walkthrough For PHP Developers

This document explains the `homelabctl` codebase from the point of view of a PHP/Symfony developer learning Go.

It is intentionally more explicit than normal production documentation. The source files now also contain idiomatic Go comments on exported types and functions, similar in spirit to PHPDoc on public classes and methods.

## How To Read Go Code

Go source files are organized by package, not by class. A package is closer to a small PHP namespace plus a directory-level module.

Common syntax translations:

| Go | PHP/Symfony mental model |
| --- | --- |
| `package config` | `namespace App\Config;`, but package names are short and directory-based |
| `import (...)` | `use ...;` imports |
| `func Load(...)` | a package-level function, similar to a static helper but idiomatic in Go |
| `type Config struct` | a DTO/value object class with public fields |
| `type Runner interface` | an interface with method signatures |
| `err != nil` | explicit error handling instead of exceptions |
| `return value, err` | Go often returns a result plus an error |
| `defer client.Close()` | "run this cleanup before the function returns" |
| `context.Context` | cancellation/deadline object passed through a call chain |
| `[]T` | slice, similar to a typed array/list |
| `map[K]V` | associative array with explicit key/value types |
| `*T` | pointer to `T`, similar to passing object references or nullable value holders |
| struct tags like ``json:"vmid"`` | serializer metadata, like Symfony Serializer annotations/attributes |

## Project Layout

```text
cmd/homelabctl/main.go
```

The CLI executable entrypoint. In Symfony terms, this is like `bin/console`, except it is compiled into the binary.

```text
internal/app/root.go
```

Builds the Cobra command tree. This is where CLI commands, flags, and handlers are registered.

```text
internal/config/config.go
```

Loads YAML configuration, expands `~`, applies defaults, and validates required fields.

```text
internal/sshx/client.go
```

Contains the SSH implementation. Other packages depend on the small `Runner` interface so tests can use fakes.

```text
internal/proxmox/lxc.go
```

Contains the Proxmox-specific LXC listing use case and parsers.

```text
internal/output/*.go
```

Contains presentation code for JSON and tables.

```text
*_test.go
```

Go tests. These live next to the package they test, like PHPUnit tests colocated with the unit under test.

## `cmd/homelabctl/main.go`

```go
package main
```

Every compiled Go program starts in package `main`. Libraries use other package names.

```go
import (
    "fmt"
    "os"

    "github.com/bartlomiejnoszka/go-homelabctl/internal/app"
)
```

The standard library imports are first. The blank line separates them from project imports. `fmt` prints formatted text, `os` gives access to stderr and exit codes, and `internal/app` builds the CLI.

```go
func main() {
```

`main` is the executable entrypoint, like PHP's first executed script.

```go
if err := app.NewRootCmd().Execute(); err != nil {
```

This creates the Cobra root command and runs it. The `if err := ...; err != nil` syntax declares `err` only for this `if` statement. In PHP, imagine:

```php
$err = $app->run();
if ($err !== null) { ... }
```

```go
fmt.Fprintln(os.Stderr, err)
os.Exit(1)
```

Write the error to stderr and exit with a non-zero status. This is command-line etiquette.

## `internal/app/root.go`

This file is the CLI composition root.

```go
package app
```

Everything in this file belongs to package `app`.

```go
import (
    "context"
    "os"
    ...
    "github.com/spf13/cobra"
)
```

`context` carries cancellation/deadline information. `os` provides stdout/stderr. Cobra is the CLI framework.

```go
func NewRootCmd() *cobra.Command
```

Returns a pointer to a Cobra command. In PHP terms, `*cobra.Command` is like returning an object reference.

```go
var configPath string
defPath, _ := config.DefaultPath()
```

`configPath` will hold the `--config` flag value. `:=` declares and assigns. `_` discards the error because the fallback can be an empty default here.

```go
rootCmd := &cobra.Command{ ... }
```

`&cobra.Command{}` creates a struct value and returns a pointer to it. This is like `new Command(...)`, but with named fields.

```go
rootCmd.PersistentFlags().StringVar(&configPath, "config", defPath, "Path to config file")
```

Registers `--config`. The `&configPath` means Cobra writes the parsed flag value into that variable.

```go
lxcCmd := &cobra.Command{Use: "lxc", Short: "LXC-related commands"}
```

Creates the command group `homelabctl lxc`.

```go
listCmd := &cobra.Command{ ... RunE: func(...) error { ... } }
```

Creates `homelabctl lxc list`. `RunE` is a callback that can return an error. This is like a Symfony command `execute()` method returning success/failure, but with explicit errors.

Inside `RunE`:

```go
jsonOut, _ := cmd.Flags().GetBool("json")
```

Reads `--json`. `_` ignores the error because the flag is defined by this command.

```go
cfg, err := config.Load(configPath)
if err != nil { return err }
```

Load config and return early on error. This pattern replaces many exception flows in Go.

```go
runner := sshx.NewSSHRunner(...)
```

Build the SSH command runner from config values.

```go
containers, err := proxmox.ListLXC(context.Background(), runner)
```

Calls the Proxmox use case. `context.Background()` is the root context when there is no request-specific context.

```go
if jsonOut { return output.WriteJSON(os.Stdout, containers) }
output.WriteLXCTable(os.Stdout, containers)
return nil
```

Choose JSON or table output. `nil` means "no error".

```go
listCmd.Flags().Bool("json", false, "Output as JSON")
lxcCmd.AddCommand(listCmd)
rootCmd.AddCommand(lxcCmd)
```

Registers flags and command hierarchy.

```go
rootCmd.SilenceUsage = true
rootCmd.SilenceErrors = true
```

Stops Cobra from printing duplicate noise. `main` owns final error printing.

## `internal/config/config.go`

This file turns YAML into typed Go data.

```go
const (
    defaultPort = 22
    defaultTimeout = 5 * time.Second
)
```

Constants are immutable values. `5 * time.Second` is a typed duration.

```go
type Config struct {
    Proxmox ProxmoxConfig `yaml:"proxmox"`
}
```

`Config` mirrors the YAML root. The backtick part is a struct tag used by `yaml.v3`.

```go
type ProxmoxConfig struct { ... }
```

Fields beginning with uppercase letters are exported, like public properties. Lowercase fields would be package-private.

```go
Timeout time.Duration `yaml:"-"`
```

The `yaml:"-"` tag tells YAML parsing to ignore this computed field.

```go
func DefaultPath() (string, error)
```

Returns two values: the path and an error. Go prefers this over throwing exceptions.

```go
home, err := os.UserHomeDir()
if err != nil { return "", fmt.Errorf(...) }
```

Get the user home directory and wrap any error with context.

```go
return filepath.Join(home, ".config", "homelabctl", "config.yaml"), nil
```

`filepath.Join` is cross-platform path joining. `nil` means no error.

```go
func Load(path string) (*Config, error)
```

Returns a pointer to `Config`, because callers should receive one loaded config object without copying it.

```go
strings.TrimSpace(path) == ""
```

Trims whitespace before checking empty strings.

```go
expandedPath, err := expandTilde(path)
```

Small helper function. Private helpers start with lowercase names.

```go
content, err := os.ReadFile(expandedPath)
```

Reads the whole file into `[]byte`, similar to PHP `file_get_contents`.

```go
errors.Is(err, os.ErrNotExist)
```

Checks whether the error is "file not found", even if it is wrapped.

```go
var cfg Config
yaml.Unmarshal(content, &cfg)
```

Declare an empty struct and pass its address so the YAML library can fill it.

```go
if err := cfg.applyDefaultsAndValidate(); err != nil { ... }
```

Method call on `Config`. The receiver is defined as `(c *Config)`, similar to `$this` but explicit.

```go
p := &c.Proxmox
```

Take a pointer to the nested config so edits update the original struct.

```go
if p.Port == 0 { p.Port = defaultPort }
```

In Go, numeric fields default to zero, so zero can mean "not provided" for config defaults.

```go
p.Timeout = time.Duration(p.TimeoutSeconds) * time.Second
```

Convert integer seconds into a typed duration.

```go
return errors.New("...")
```

Create a simple error value. Use `fmt.Errorf` when formatting or wrapping.

```go
if path == "~" || strings.HasPrefix(path, "~/") { ... }
```

Manual shell-style tilde expansion. Go does not expand `~` automatically.

## `internal/sshx/client.go`

This file is the adapter around `golang.org/x/crypto/ssh`.

```go
type Runner interface {
    Run(ctx context.Context, command string) (string, error)
}
```

An interface is satisfied implicitly. Any type with this method is a `Runner`; no `implements` keyword is needed.

```go
type SSHRunner struct { ... }
```

Concrete implementation. Fields are lowercase, so only this package can access them.

```go
dialerFunc func(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error)
```

A function field. This makes tests possible because it can be swapped if needed.

```go
func NewSSHRunner(...) *SSHRunner
```

Factory function. Go does not require constructors; `NewX` is a naming convention.

```go
return &SSHRunner{host: host, ...}
```

Struct literal with named fields. `&` returns a pointer.

```go
func (r *SSHRunner) Run(...)
```

Method with receiver `r`. In PHP terms, `r` is like `$this`, but it is explicitly named.

```go
signer, err := signerFromPrivateKey(r.keyPath)
```

Load and parse the private key before connecting.

```go
cfg := &ssh.ClientConfig{ ... }
```

Build SSH client settings. The `//nolint:gosec` comment acknowledges insecure host-key checking for this MVP.

```go
client, err := r.dialerFunc("tcp", fmt.Sprintf("%s:%d", r.host, r.port), cfg)
```

Open a TCP SSH connection to `host:port`.

```go
defer client.Close()
```

Ensure the connection is closed before `Run` returns.

```go
session, err := client.NewSession()
defer session.Close()
```

Each remote command runs in an SSH session. This is separate from the TCP connection object.

```go
type result struct { out []byte; err error }
```

Small local struct used only inside this function.

```go
ch := make(chan result, 1)
go func() { ... }()
```

Start a goroutine and send the command result through a channel. This lets the `select` below react to cancellation.

```go
select { case <-ctx.Done(): ... case res := <-ch: ... }
```

Wait for whichever happens first: context cancellation or command completion.

```go
session.CombinedOutput(command)
```

Runs the command and returns stdout plus stderr. This is useful for error messages.

## `internal/proxmox/lxc.go`

This is the main domain/use-case file.

```go
type LXCContainer struct { ... }
```

DTO for output. JSON tags define API field names. Pointer fields like `*float64` let us distinguish "zero" from "unknown".

```go
type LXCConfig struct { ... }
type LXCDiskUsage struct { ... }
```

Intermediate parsed values. They are smaller than `LXCContainer` because they come from one specific command.

```go
const (... markers ...)
```

Marker strings wrap sections in the remote shell output. This is safer than guessing where one command's output ends and another begins.

```go
func ListLXC(ctx context.Context, runner Runner) ([]LXCContainer, error)
```

The orchestration function:

1. Run the batched remote command.
2. Split `pct list` from details.
3. Parse containers.
4. Parse config/IP/disk details.
5. Merge everything by VMID.

```go
stdout, err := runner.Run(ctx, buildLXCListCommand())
```

The Proxmox package does not know whether this is real SSH or a fake test runner.

```go
listOut, detailsOut, err := SplitLXCListBatch(stdout)
containers, err := ParsePCTList(listOut)
configs, liveIPs, diskUsages, err := ParseLXCDetails(detailsOut)
```

Each parser has one responsibility, which keeps tests focused.

```go
for i := range containers { ... }
```

Iterate by index because we update the slice elements in place.

```go
cfg, ok := configs[containers[i].VMID]
if !ok { return nil, fmt.Errorf(...) }
```

Map lookup returns value plus boolean. This is like checking `array_key_exists`.

```go
if ips := liveIPs[containers[i].VMID]; len(ips) > 0 { ... }
```

Short variable declaration scoped to the `if`.

```go
func buildLXCListCommand() string
```

Builds a shell script string. It runs one SSH command instead of many slow SSH round trips.

The script:

```sh
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
```

Creates temporary files on Proxmox and guarantees cleanup.

```sh
list="$(pct list)"
```

Captures the base container list once.

```sh
awk 'NR > 1 { print $1 " " $2 }'
```

Skips the header and emits `vmid status`.

```sh
( ... ) &
```

Runs per-container work in parallel on the remote host.

```sh
pct config "$vmid"
pct exec "$vmid" -- df -B1 -P /
pct exec "$vmid" -- ip -4 -o addr show scope global
```

Gets config, root filesystem usage, and live IPs.

```sh
cat "$tmpdir/$vmid.out"
```

Prints temp files later in `pct list` order, so output stays deterministic.

Parser helpers:

```go
strings.Fields(line)
```

Split by any whitespace. Useful for command-line table output.

```go
strconv.Atoi(...)
strconv.ParseFloat(...)
```

Convert strings into typed numbers.

```go
strings.Cut(line, ":")
```

Split once into `before`, `after`, and `ok`, similar to a safer `explode(':', $line, 2)`.

```go
math.Round(freeGB*100) / 100
```

Round to two decimals for stable JSON/table output.

```go
strings.HasPrefix(key, "net")
```

Proxmox network config keys look like `net0`, `net1`, and so on.

## `internal/output`

```go
func WriteJSON(w io.Writer, v any) error
```

`any` means any type, like PHP `mixed`. `io.Writer` means the function can write to stdout, a file, or a test buffer.

```go
enc := json.NewEncoder(w)
enc.SetIndent("", "  ")
return enc.Encode(v)
```

Create a streaming JSON encoder, configure pretty formatting, and encode the value.

```go
func WriteLXCTable(w io.Writer, items []proxmox.LXCContainer)
```

Accepts a writer and a typed slice of containers.

```go
tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
```

`tabwriter` aligns tab-separated columns for terminal output.

```go
pid := "-"
if c.PID != nil { pid = fmt.Sprintf("%d", *c.PID) }
```

Pointers can be nil. Dereference with `*c.PID` only after checking nil.

```go
strings.Join(c.IPAddresses, ",")
```

Equivalent to PHP `implode(',', $addresses)`.

```go
fmt.Fprintf(tw, "...", values...)
```

Formatted write to the table writer, like `printf` targeting a stream.

## Tests

Go tests use the standard `testing` package and functions named `TestXxx`.

```go
func TestSomething(t *testing.T)
```

This is like a PHPUnit test method, but it is just a function.

```go
t.Fatalf("message: %v", err)
```

Fail the test immediately with a formatted message.

```go
fakeRunner struct { outputs map[string]string; errs map[string]error; calls []string }
```

The fake runner replaces real SSH. This is the Go version of mocking an interface, but without a mocking framework.

```go
var b bytes.Buffer
```

An in-memory writer used to test output functions without touching stdout.

```go
strings.Contains(s, "...")
```

Simple assertion strategy for output tests.

## Why There Is No Service Container

Symfony often wires services through a container. This app uses explicit construction:

```go
runner := sshx.NewSSHRunner(...)
containers, err := proxmox.ListLXC(ctx, runner)
```

The dependency is passed directly. This keeps the program small and makes tests straightforward.

## Why Errors Are Returned Instead Of Thrown

Go treats errors as values:

```go
value, err := doThing()
if err != nil {
    return err
}
```

This can feel noisy coming from PHP exceptions, but it makes the failure path visible and local.

## Why So Many Small Packages

Each package has one job:

- `config`: read config
- `sshx`: execute commands
- `proxmox`: understand Proxmox output
- `output`: render output
- `app`: wire the CLI

That is the Go version of keeping Symfony services focused, but without building a framework around them.
