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
	"encoding/csv"
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
	ModeCSV   Mode = "csv"
	ModeTSV   Mode = "tsv"
)

// Printer handles CLI output.
type Printer struct {
	Mode          Mode
	Stdout        io.Writer
	Stderr        io.Writer
	IsTTY         bool
	IndentJSON    bool
	TruncateTable bool
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
		if p.IndentJSON {
			enc.SetIndent("", "  ")
		}
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
		if p.IndentJSON {
			enc.SetIndent("", "  ")
		}
		return enc.Encode(data)
	default:
		enc := json.NewEncoder(p.Stdout)
		if p.IndentJSON {
			enc.SetIndent("", "  ")
		}
		return enc.Encode(data)
	}
}

// PrintTable renders a table with headers and rows.
func (p *Printer) PrintTable(headers []string, rows [][]string) error {
	if p.Mode == ModeJSON || p.Mode == ModeYAML {
		// Convert to simple objects for JSON/YAML output.
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
	if p.Mode == ModeCSV {
		return p.PrintCSV(headers, rows)
	}
	if p.Mode == ModeTSV {
		return p.PrintTSV(headers, rows)
	}

	table := tablewriter.NewWriter(p.Stdout)
	table.Header(headers)
	for _, row := range rows {
		iface := make([]any, len(row))
		for i, v := range row {
			if p.TruncateTable && len(v) > 40 {
				v = v[:37] + "..."
			}
			iface[i] = v
		}
		_ = table.Append(iface...)
	}
	return table.Render()
}

// PrintError renders a structured error message.
func (p *Printer) PrintError(msg string, suggestions []string) {
	switch p.Mode {
	case ModeJSON:
		payload := map[string]any{"error": msg}
		if len(suggestions) > 0 {
			payload["suggestions"] = suggestions
		}
		_ = json.NewEncoder(p.Stderr).Encode(payload)
	case ModeYAML:
		payload := map[string]any{"error": msg}
		if len(suggestions) > 0 {
			payload["suggestions"] = suggestions
		}
		_ = yaml.NewEncoder(p.Stderr).Encode(payload)
	default:
		_, _ = fmt.Fprintf(p.Stderr, "error: %s\n", msg)
		if len(suggestions) > 0 {
			_, _ = fmt.Fprintln(p.Stderr)
			_, _ = fmt.Fprintln(p.Stderr, "Try:")
			for _, s := range suggestions {
				_, _ = fmt.Fprintf(p.Stderr, "  %s\n", s)
			}
		}
	}
}

// PrintSuccess prints a success message.
func (p *Printer) PrintSuccess(msg string) {
	switch p.Mode {
	case ModeJSON:
		_ = json.NewEncoder(p.Stdout).Encode(map[string]string{"message": msg})
	case ModeYAML:
		_ = yaml.NewEncoder(p.Stdout).Encode(map[string]string{"message": msg})
	default:
		_, _ = fmt.Fprintf(p.Stdout, "✓ %s\n", msg)
	}
}

// PrintCSV renders headers and rows as CSV.
func (p *Printer) PrintCSV(headers []string, rows [][]string) error {
	w := csv.NewWriter(p.Stdout)
	if err := w.Write(headers); err != nil {
		return err
	}
	for _, row := range rows {
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}

// PrintTSV renders headers and rows as tab-separated values.
func (p *Printer) PrintTSV(headers []string, rows [][]string) error {
	_, err := fmt.Fprintln(p.Stdout, strings.Join(headers, "\t"))
	if err != nil {
		return err
	}
	for _, row := range rows {
		_, err := fmt.Fprintln(p.Stdout, strings.Join(row, "\t"))
		if err != nil {
			return err
		}
	}
	return nil
}

// PrintProgress prints a simple text progress bar.
func (p *Printer) PrintProgress(current, total int, label string) {
	if total <= 0 {
		return
	}
	percent := (current * 100) / total
	filled := (current * 10) / total
	bar := strings.Repeat("#", filled) + strings.Repeat("-", 10-filled)
	_, _ = fmt.Fprintf(p.Stdout, "[%s] %d%% %s\n", bar, percent, label)
}

// PrintWarning prints a warning message.
func (p *Printer) PrintWarning(msg string) {
	switch p.Mode {
	case ModeJSON:
		_ = json.NewEncoder(p.Stdout).Encode(map[string]string{"warning": msg})
	case ModeYAML:
		_ = yaml.NewEncoder(p.Stdout).Encode(map[string]string{"warning": msg})
	default:
		if p.IsTTY {
			_, _ = fmt.Fprintf(p.Stdout, "\x1b[33mwarning: \x1b[0m%s\n", msg)
			return
		}
		_, _ = fmt.Fprintf(p.Stdout, "warning: %s\n", msg)
	}
}

// PrintInfo prints an informational message.
func (p *Printer) PrintInfo(msg string) {
	switch p.Mode {
	case ModeJSON:
		_ = json.NewEncoder(p.Stdout).Encode(map[string]string{"info": msg})
	case ModeYAML:
		_ = yaml.NewEncoder(p.Stdout).Encode(map[string]string{"info": msg})
	default:
		if p.IsTTY {
			_, _ = fmt.Fprintf(p.Stdout, "\x1b[34minfo: \x1b[0m%s\n", msg)
			return
		}
		_, _ = fmt.Fprintf(p.Stdout, "info: %s\n", msg)
	}
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
	case "csv":
		return ModeCSV, nil
	case "tsv":
		return ModeTSV, nil
	default:
		return "", fmt.Errorf("unknown output mode: %s", s)
	}
}
