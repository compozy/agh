package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
)

// OutputFormat controls CLI output rendering.
type OutputFormat string

const (
	// OutputHuman renders human-readable tables and sections.
	OutputHuman OutputFormat = "human"
	// OutputJSON renders raw JSON payloads.
	OutputJSON OutputFormat = "json"
	// OutputJSONL renders newline-delimited JSON for streaming-style commands.
	OutputJSONL OutputFormat = "jsonl"
	// OutputToon renders a compact LLM-friendly TOON-like text document.
	OutputToon OutputFormat = "toon"
)

type outputBundle struct {
	jsonValue any
	human     func() (string, error)
	toon      func() (string, error)
}

func listBundle[T any](
	jsonValue any,
	items []T,
	humanTitle string,
	humanHeaders []string,
	toonName string,
	toonFields []string,
	humanRow func(T) []string,
	toonRow func(T) []string,
) outputBundle {
	return outputBundle{
		jsonValue: jsonValue,
		human: func() (string, error) {
			if humanRow == nil {
				return "", errors.New("cli: human list row renderer is required")
			}
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				rows = append(rows, humanRow(item))
			}
			return renderHumanTable(humanTitle, humanHeaders, rows), nil
		},
		toon: func() (string, error) {
			if toonRow == nil {
				return "", errors.New("cli: toon list row renderer is required")
			}
			rows := make([][]string, 0, len(items))
			for _, item := range items {
				rows = append(rows, toonRow(item))
			}
			return renderToonArray(toonName, toonFields, rows), nil
		},
	}
}

type keyValue struct {
	Label string
	Value string
}

func resolveOutputFormat(cmd *cobra.Command) (OutputFormat, error) {
	if cmd == nil {
		return "", errors.New("cli: command is required")
	}

	value, err := cmd.Flags().GetString(outputFlagName)
	if err != nil {
		return "", fmt.Errorf("cli: read output flag: %w", err)
	}

	switch OutputFormat(strings.ToLower(strings.TrimSpace(value))) {
	case "", OutputHuman:
		return OutputHuman, nil
	case OutputJSON:
		return OutputJSON, nil
	case OutputJSONL:
		return OutputJSONL, nil
	case OutputToon:
		return OutputToon, nil
	default:
		return "", fmt.Errorf("cli: invalid output format %q", value)
	}
}

func writeCommandOutput(cmd *cobra.Command, bundle outputBundle) error {
	mode, err := resolveOutputFormat(cmd)
	if err != nil {
		return err
	}

	switch mode {
	case OutputJSON:
		return writeJSON(cmd, bundle.jsonValue)
	case OutputJSONL:
		return errors.New("cli: jsonl output is only supported by streaming commands")
	case OutputToon:
		if bundle.toon == nil {
			return errors.New("cli: toon formatter is required")
		}
		rendered, err := bundle.toon()
		if err != nil {
			return err
		}
		return writeRawCommandOutput(cmd, rendered)
	default:
		if bundle.human == nil {
			return errors.New("cli: human formatter is required")
		}
		rendered, err := bundle.human()
		if err != nil {
			return err
		}
		return writeRawCommandOutput(cmd, rendered)
	}
}

func writeJSONLine(cmd *cobra.Command, value any) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func writeJSON(cmd *cobra.Command, value any) error {
	encoder := json.NewEncoder(cmd.OutOrStdout())
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeRawCommandOutput(cmd *cobra.Command, text string) error {
	writer := cmd.OutOrStdout()
	if strings.HasSuffix(text, "\n") {
		_, err := fmt.Fprint(writer, text)
		return err
	}
	_, err := fmt.Fprintln(writer, text)
	return err
}

func renderHumanSection(title string, rows []keyValue) string {
	var buffer bytes.Buffer
	if title != "" {
		fmt.Fprintf(&buffer, "%s\n", title)
		fmt.Fprintf(&buffer, "%s\n", strings.Repeat("=", len(title)))
	}

	writer := tabwriter.NewWriter(&buffer, 0, 0, 2, ' ', 0)
	for _, row := range rows {
		_, _ = fmt.Fprintf(writer, "%s:\t%s\n", row.Label, row.Value)
	}
	_ = writer.Flush()

	return strings.TrimRight(buffer.String(), "\n")
}

func renderHumanTable(title string, headers []string, rows [][]string) string {
	var buffer bytes.Buffer
	if title != "" {
		fmt.Fprintf(&buffer, "%s\n", title)
		fmt.Fprintf(&buffer, "%s\n", strings.Repeat("=", len(title)))
	}
	if len(headers) == 0 {
		return strings.TrimRight(buffer.String(), "\n")
	}

	writer := tabwriter.NewWriter(&buffer, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, strings.Join(headers, "\t"))
	separators := make([]string, 0, len(headers))
	for _, header := range headers {
		separators = append(separators, strings.Repeat("-", max(3, len(header))))
	}
	_, _ = fmt.Fprintln(writer, strings.Join(separators, "\t"))

	if len(rows) == 0 {
		_, _ = fmt.Fprintln(writer, "(empty)")
	} else {
		for _, row := range rows {
			_, _ = fmt.Fprintln(writer, strings.Join(row, "\t"))
		}
	}
	_ = writer.Flush()

	return strings.TrimRight(buffer.String(), "\n")
}

func renderHumanBlocks(blocks ...string) string {
	parts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		trimmed := strings.TrimSpace(block)
		if trimmed == "" {
			continue
		}
		parts = append(parts, trimmed)
	}
	return strings.Join(parts, "\n\n")
}

func renderToonObject(name string, fields []string, values []string) string {
	var builder strings.Builder
	builder.WriteString(name)
	builder.WriteByte('{')
	builder.WriteString(strings.Join(fields, ","))
	builder.WriteString("}:\n  ")
	writeToonValues(&builder, values)
	return builder.String()
}

func renderToonArray(name string, fields []string, rows [][]string) string {
	var builder strings.Builder
	builder.WriteString(name)
	builder.WriteByte('[')
	builder.WriteString(strconv.Itoa(len(rows)))
	builder.WriteByte(']')
	builder.WriteByte('{')
	builder.WriteString(strings.Join(fields, ","))
	builder.WriteString("}:")
	if len(rows) == 0 {
		builder.WriteString("\n  (empty)")
		return builder.String()
	}
	for _, row := range rows {
		builder.WriteString("\n  ")
		writeToonValues(&builder, row)
	}
	return builder.String()
}

func writeToonValues(builder *strings.Builder, values []string) {
	for i, value := range values {
		if i > 0 {
			builder.WriteByte(',')
		}
		builder.WriteString(toonValue(value))
	}
}

func toonValue(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return `""`
	}
	if strings.ContainsAny(trimmed, ",\n\r\t\"") {
		return strconv.Quote(trimmed)
	}
	return trimmed
}

func compactJSON(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var buffer bytes.Buffer
	if err := json.Compact(&buffer, raw); err != nil {
		return strings.TrimSpace(string(raw))
	}
	return buffer.String()
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

func formatAge(now func() time.Time, then time.Time) string {
	if then.IsZero() {
		return ""
	}
	current := time.Now().UTC()
	if now != nil {
		current = now().UTC()
	}
	delta := max(current.Sub(then.UTC()), 0)

	switch {
	case delta < time.Minute:
		return fmt.Sprintf("%ds", int(delta.Seconds()))
	case delta < time.Hour:
		return fmt.Sprintf("%dm", int(delta.Minutes()))
	case delta < 24*time.Hour:
		return fmt.Sprintf("%dh", int(delta.Hours()))
	default:
		return fmt.Sprintf("%dd", int(delta.Hours()/24))
	}
}

func stringOrDash(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "--"
	}
	return trimmed
}

func intOrDash(value int) string {
	if value <= 0 {
		return "--"
	}
	return strconv.Itoa(value)
}

func int64OrDash(value int64) string {
	if value <= 0 {
		return "--"
	}
	return strconv.FormatInt(value, 10)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
