package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/nbw/firehose/internal/client"
)

type Printer interface {
	Tap(t client.Tap, raw []byte)
	Taps(ts []client.Tap, raw []byte)
	TapCreated(res client.CreateTapResult, raw []byte)
	Rule(r client.Rule, raw []byte)
	Rules(rs []client.Rule, raw []byte)
	Deleted(kind, id string)
}

func New(out, errOut io.Writer, jsonMode bool) Printer {
	if jsonMode {
		return &jsonPrinter{out: out, errOut: errOut}
	}
	return &prettyPrinter{out: out, errOut: errOut}
}

type jsonPrinter struct {
	out    io.Writer
	errOut io.Writer
}

func (p *jsonPrinter) emit(raw []byte, fallback any) {
	if len(raw) > 0 {
		p.out.Write(raw)
		if len(raw) == 0 || raw[len(raw)-1] != '\n' {
			p.out.Write([]byte{'\n'})
		}
		return
	}
	enc := json.NewEncoder(p.out)
	enc.SetIndent("", "  ")
	_ = enc.Encode(fallback)
}

func (p *jsonPrinter) Tap(t client.Tap, raw []byte)                  { p.emit(raw, t) }
func (p *jsonPrinter) Taps(ts []client.Tap, raw []byte)              { p.emit(raw, ts) }
func (p *jsonPrinter) TapCreated(res client.CreateTapResult, raw []byte) {
	p.emit(raw, res)
	if res.Token != "" {
		fmt.Fprintln(p.errOut, "WARNING: tap token shown only once. Store it securely.")
	}
}
func (p *jsonPrinter) Rule(r client.Rule, raw []byte)     { p.emit(raw, r) }
func (p *jsonPrinter) Rules(rs []client.Rule, raw []byte) { p.emit(raw, rs) }
func (p *jsonPrinter) Deleted(kind, id string) {
	enc := json.NewEncoder(p.out)
	_ = enc.Encode(map[string]any{"deleted": true, "kind": kind, "id": id})
}

type prettyPrinter struct {
	out    io.Writer
	errOut io.Writer
}

func (p *prettyPrinter) Tap(t client.Tap, _ []byte) {
	printTap(p.out, t)
}

func (p *prettyPrinter) Taps(ts []client.Tap, _ []byte) {
	if len(ts) == 0 {
		fmt.Fprintln(p.out, "No taps.")
		return
	}
	tapsTable(p.out, ts)
}

func (p *prettyPrinter) TapCreated(res client.CreateTapResult, _ []byte) {
	fmt.Fprintf(p.out, "Tap created: %s\n", res.Tap.ID)
	if res.Tap.Name != "" {
		fmt.Fprintf(p.out, "Name:        %s\n", res.Tap.Name)
	}
	if res.Token != "" {
		fmt.Fprintf(p.out, "Token:       %s\n", res.Token)
		fmt.Fprintln(p.errOut, "WARNING: tap token shown only once. Store it securely.")
	}
}

func (p *prettyPrinter) Rule(r client.Rule, _ []byte) {
	printRule(p.out, r)
}

func (p *prettyPrinter) Rules(rs []client.Rule, _ []byte) {
	if len(rs) == 0 {
		fmt.Fprintln(p.out, "No rules.")
		return
	}
	rulesTable(p.out, rs)
}

func (p *prettyPrinter) Deleted(kind, id string) {
	fmt.Fprintf(p.out, "Deleted %s %s\n", kind, id)
}

func printTap(w io.Writer, t client.Tap) {
	fmt.Fprintf(w, "ID:           %s\n", t.ID)
	fmt.Fprintf(w, "Name:         %s\n", t.Name)
	if t.TokenPrefix != "" {
		fmt.Fprintf(w, "Token prefix: %s\n", t.TokenPrefix)
	}
	if t.Token != "" {
		fmt.Fprintf(w, "Token:        %s\n", t.Token)
	}
	if t.RulesCount > 0 {
		fmt.Fprintf(w, "Rules:        %d\n", t.RulesCount)
	}
	if t.LastUsedAt != nil && *t.LastUsedAt != "" {
		fmt.Fprintf(w, "Last used:    %s\n", *t.LastUsedAt)
	}
	if t.CreatedAt != "" {
		fmt.Fprintf(w, "Created:      %s\n", t.CreatedAt)
	}
}

func printRule(w io.Writer, r client.Rule) {
	fmt.Fprintf(w, "ID:      %s\n", r.ID)
	fmt.Fprintf(w, "Value:   %s\n", r.Value)
	if r.Tag != "" {
		fmt.Fprintf(w, "Tag:     %s\n", r.Tag)
	}
	if r.NSFW != nil {
		fmt.Fprintf(w, "NSFW:    %t\n", *r.NSFW)
	}
	if r.Quality != nil {
		fmt.Fprintf(w, "Quality: %t\n", *r.Quality)
	}
}
