// Package omabittheme owns Omarchy/Talon-specific appearance behavior.
package omabittheme

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
	"unicode/utf16"
)

const ThemeID = "omarchy-urbit-theme"

type Palette struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Dark       bool   `json:"dark"`
	Primary    string `json:"primary"`
	Secondary  string `json:"secondary"`
	Tertiary   string `json:"tertiary"`
	Background string `json:"background"`
	Surface    string `json:"surface"`
}

var colorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func (p Palette) Validate() error {
	if p.ID != ThemeID || len(p.Name) == 0 || len(p.Name) > 256 || strings.ContainsAny(p.Name, "\r\n\x00") {
		return errors.New("invalid theme identity or name")
	}
	for _, color := range []string{p.Primary, p.Secondary, p.Tertiary, p.Background, p.Surface} {
		if !colorPattern.MatchString(color) {
			return errors.New("invalid six-digit theme color")
		}
	}
	return nil
}
func (p Palette) Fingerprint() string {
	data, _ := json.Marshal(p)
	var fields map[string]any
	json.Unmarshal(data, &fields)
	var buffer bytes.Buffer
	encoder := json.NewEncoder(&buffer)
	encoder.SetEscapeHTML(false)
	_ = encoder.Encode(fields)
	var canonical strings.Builder
	for _, r := range strings.TrimSuffix(buffer.String(), "\n") {
		if r < 128 {
			canonical.WriteRune(r)
		} else {
			for _, unit := range utf16.Encode([]rune{r}) {
				fmt.Fprintf(&canonical, "\\u%04x", unit)
			}
		}
	}
	hash := sha256.Sum256([]byte(canonical.String()))
	return hex.EncodeToString(hash[:])
}
func Resolve(ctx context.Context) (Palette, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "omarchy", "theme", "color", "--all")
	cmd.Stderr = io.Discard
	var output limitedOutput
	cmd.Stdout = &output
	if err := cmd.Run(); err != nil {
		return Palette{}, errors.New("Omarchy palette resolver is unavailable")
	}
	data := output.Bytes()
	colors := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		key, value, ok := strings.Cut(line, "\t")
		if !ok || colors[key] != "" {
			return Palette{}, errors.New("invalid palette resolver output")
		}
		colors[key] = value
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return Palette{}, err
	}
	f, err := os.Open(filepath.Join(home, ".local/state/omarchy/current/theme.name"))
	if err != nil {
		return Palette{}, errors.New("current Omarchy theme name unavailable")
	}
	defer f.Close()
	name, err := io.ReadAll(io.LimitReader(f, 1025))
	if err != nil || len(name) > 1024 {
		return Palette{}, errors.New("invalid current theme name")
	}
	fallback := func(keys ...string) string {
		for _, key := range keys {
			if colors[key] != "" {
				return strings.ToUpper(colors[key])
			}
		}
		return ""
	}
	p := Palette{ID: ThemeID, Name: strings.TrimSpace(string(name)), Dark: colors["mode"] == "dark", Primary: fallback("accent", "blue", "foreground"), Secondary: fallback("blue", "accent", "foreground"), Tertiary: fallback("green", "blue", "accent", "foreground"), Background: fallback("background"), Surface: fallback("lighter_background", "background")}
	if colors["mode"] != "dark" && colors["mode"] != "light" {
		return p, errors.New("invalid Omarchy theme mode")
	}
	return p, p.Validate()
}
