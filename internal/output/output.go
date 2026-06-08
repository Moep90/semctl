// Copyright 2026 The semctl authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package output

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/olekukonko/tablewriter"
	"gopkg.in/yaml.v3"
)

// ansiEscape matches ANSI escape sequences (CSI, OSC, and other control sequences).
var ansiEscape = regexp.MustCompile(`\x1b\[[0-9;]*[A-Za-z]|\x1b\][^\x07]*(?:\x07|\x1b\\)|\x1b[()%@><=]`)

// Mode is the output format.
type Mode string

const (
	ModeTable Mode = "table"
	ModeJSON  Mode = "json"
	ModeYAML  Mode = "yaml"
	ModeText  Mode = "text"
)

// Printer handles CLI output.
type Printer struct {
	Mode   Mode
	Stdout io.Writer
	Stderr io.Writer
	IsTTY  bool
}

// New creates a default printer.
func New(mode Mode) *Printer {
	isTTY := isTerminal(os.Stdout)
	if mode == "" {
		if isTTY {
			mode = ModeTable
		} else {
			mode = ModeJSON
		}
	}
	return &Printer{
		Mode:   mode,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		IsTTY:  isTTY,
	}
}

// Print outputs data according to the current mode.
func (p *Printer) Print(data any) error {
	switch p.Mode {
	case ModeJSON:
		enc := json.NewEncoder(p.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	case ModeYAML:
		return yaml.NewEncoder(p.Stdout).Encode(data)
	case ModeText:
		if s, ok := data.(string); ok {
			_, err := fmt.Fprintln(p.Stdout, s)
			return err
		}
		_, err := fmt.Fprintln(p.Stdout, data)
		return err
	case ModeTable:
		// Tables require specific data shapes; PrintTable should be used instead.
		enc := json.NewEncoder(p.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	default:
		enc := json.NewEncoder(p.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(data)
	}
}

// PrintTable renders a table with headers and rows.
func (p *Printer) PrintTable(headers []string, rows [][]string) error {
	if p.Mode == ModeJSON {
		// Convert to simple objects for JSON output.
		var out []map[string]string
		for _, row := range rows {
			m := make(map[string]string, len(headers))
			for i, h := range headers {
				if i < len(row) {
					m[h] = row[i]
				}
			}
			out = append(out, m)
		}
		return p.Print(out)
	}
	if p.Mode == ModeYAML {
		var out []map[string]string
		for _, row := range rows {
			m := make(map[string]string, len(headers))
			for i, h := range headers {
				if i < len(row) {
					m[h] = row[i]
				}
			}
			out = append(out, m)
		}
		return p.Print(out)
	}

	table := tablewriter.NewWriter(p.Stdout)
	table.Header(headers)
	for _, row := range rows {
		iface := make([]any, len(row))
		for i, v := range row {
			iface[i] = v
		}
		_ = table.Append(iface...)
	}
	return table.Render()
}

// PrintError renders a structured error message.
func (p *Printer) PrintError(msg string, suggestions []string) {
	_, _ = fmt.Fprintf(p.Stderr, "error: %s\n", msg)
	if len(suggestions) > 0 {
		_, _ = fmt.Fprintln(p.Stderr)
		_, _ = fmt.Fprintln(p.Stderr, "Try:")
		for _, s := range suggestions {
			_, _ = fmt.Fprintf(p.Stderr, "  %s\n", s)
		}
	}
}

// PrintSuccess prints a success message.
func (p *Printer) PrintSuccess(msg string) {
	_, _ = fmt.Fprintf(p.Stdout, "✓ %s\n", msg)
}

func isTerminal(f *os.File) bool {
	stat, err := f.Stat()
	if err != nil {
		return false
	}
	return (stat.Mode() & os.ModeCharDevice) == os.ModeCharDevice
}

// SanitizeANSI removes ANSI escape sequences from a string.
func SanitizeANSI(s string) string {
	return ansiEscape.ReplaceAllString(s, "")
}

// ParseMode converts a string to a Mode.
func ParseMode(s string) (Mode, error) {
	switch strings.ToLower(s) {
	case "table":
		return ModeTable, nil
	case "json":
		return ModeJSON, nil
	case "yaml":
		return ModeYAML, nil
	case "text":
		return ModeText, nil
	default:
		return "", fmt.Errorf("unknown output mode: %s", s)
	}
}
