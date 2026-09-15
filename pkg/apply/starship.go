package apply

import (
	"bytes"
	_ "embed"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"

	"github.com/matheme/pkg"
)

//go:embed templates/starship.toml.tmpl
var starshipTemplate string

type starshipColors struct {
	S1, S1FG string
	S2, S2FG string
	S3, S3FG string
	S4, S4FG string
	S5, S5FG string
}

type rgb struct{ r, g, b float64 }

func parseHex(s string) (rgb, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return rgb{}, fmt.Errorf("invalid hex color %q", s)
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return rgb{}, fmt.Errorf("invalid hex color %q: %w", s, err)
	}
	return rgb{
		r: float64(v>>16&0xff) / 255,
		g: float64(v>>8&0xff) / 255,
		b: float64(v&0xff) / 255,
	}, nil
}

func (c rgb) hex() string {
	toByte := func(x float64) int {
		x = math.Max(0, math.Min(1, x))
		return int(math.Round(x * 255))
	}
	return fmt.Sprintf("#%02x%02x%02x", toByte(c.r), toByte(c.g), toByte(c.b))
}

func (c rgb) linear() rgb {
	toLinear := func(x float64) float64 {
		if x <= 0.04045 {
			return x / 12.92
		}
		return math.Pow((x+0.055)/1.055, 2.4)
	}
	return rgb{toLinear(c.r), toLinear(c.g), toLinear(c.b)}
}

func (c rgb) gamma() rgb {
	toGamma := func(x float64) float64 {
		if x <= 0.0031308 {
			return x * 12.92
		}
		return 1.055*math.Pow(x, 1/2.4) - 0.055
	}
	return rgb{toGamma(c.r), toGamma(c.g), toGamma(c.b)}
}

func mix(a, b rgb, t float64) rgb {
	la, lb := a.linear(), b.linear()
	return rgb{
		r: la.r*(1-t) + lb.r*t,
		g: la.g*(1-t) + lb.g*t,
		b: la.b*(1-t) + lb.b*t,
	}.gamma()
}

func (c rgb) hsl() (h, s, l float64) {
	max := math.Max(c.r, math.Max(c.g, c.b))
	min := math.Min(c.r, math.Min(c.g, c.b))
	l = (max + min) / 2
	if max == min {
		return 0, 0, l
	}
	d := max - min
	if l > 0.5 {
		s = d / (2 - max - min)
	} else {
		s = d / (max + min)
	}
	switch max {
	case c.r:
		h = (c.g - c.b) / d
		if c.g < c.b {
			h += 6
		}
	case c.g:
		h = (c.b-c.r)/d + 2
	default:
		h = (c.r-c.g)/d + 4
	}
	return h * 60, s, l
}

func hslRGB(h, s, l float64) rgb {
	if s == 0 {
		return rgb{l, l, l}
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	hue := func(t float64) float64 {
		if t < 0 {
			t++
		}
		if t > 1 {
			t--
		}
		switch {
		case t < 1.0/6:
			return p + (q-p)*6*t
		case t < 1.0/2:
			return q
		case t < 2.0/3:
			return p + (q-p)*(2.0/3-t)*6
		default:
			return p
		}
	}
	return rgb{hue(h/360 + 1.0/3), hue(h / 360), hue(h/360 - 1.0/3)}
}

func relLum(c rgb) float64 {
	l := c.linear()
	return 0.2126*l.r + 0.7152*l.g + 0.0722*l.b
}

func contrast(a, b rgb) float64 {
	la, lb := relLum(a), relLum(b)
	if la < lb {
		la, lb = lb, la
	}
	return (la + 0.05) / (lb + 0.05)
}

func pickText(block rgb, start rgb, target float64, preferLight bool) rgb {
	dirs := []rgb{{0, 0, 0}, {1, 1, 1}}
	if preferLight {
		dirs = []rgb{{1, 1, 1}, {0, 0, 0}}
	}
	var hit rgb
	best, bestT := false, math.Inf(1)
	fallback, fallbackContrast := block, 0.0
	for _, dir := range dirs {
		for t := 0.0; t <= 1.0001; t += 0.005 {
			c := mix(start, dir, t)
			r := contrast(c, block)
			if r > fallbackContrast {
				fallback, fallbackContrast = c, r
			}
			if r >= target && t < bestT {
				hit, bestT, best = c, t, true
				break
			}
		}
	}
	if !best {
		return fallback
	}
	return hit
}

func buildStarshipColors(theme *pkg.Theme) (*starshipColors, error) {
	bg, err := parseHex(theme.Base16["base00"])
	if err != nil {
		return nil, fmt.Errorf("failed to read base00: %w", err)
	}

	accentHex := theme.Accent
	if accentHex == "" {
		accentHex = theme.Base16["base0D"]
	}
	if accentHex == "" {
		accentHex = theme.Base16["base08"]
	}
	accent, err := parseHex(accentHex)
	if err != nil {
		return nil, fmt.Errorf("failed to read accent color: %w", err)
	}

	bgH, bgS, bgL := bg.hsl()
	acH, acS, acL := accent.hsl()

	var s1, s3, s4, s5 rgb
	if theme.Type == "light" {
		s1 = hslRGB(acH, math.Min(acS, 0.40), math.Max(acL*0.65, 0.10))
		s3 = mix(hslRGB(bgH, math.Min(bgS, 0.20), math.Max(bgL-0.16, 0.30)), accent, 0.12)
		s4 = mix(hslRGB(bgH, math.Min(bgS, 0.18), math.Max(bgL-0.05, 0.30)), accent, 0.10)
		s5 = hslRGB(bgH, bgS, math.Min(bgL+(1-bgL)*0.5, 0.995))
	} else {
		s1 = hslRGB(acH, math.Min(acS, 0.40), math.Min(acL+(1-acL)*0.40, 0.88))
		s3 = mix(hslRGB(bgH, math.Min(bgS, 0.28), math.Min(bgL+0.08, 0.95)), accent, 0.12)
		s4 = hslRGB(bgH, bgS, math.Max(bgL*0.90, 0.04))
		s5 = hslRGB(bgH, bgS, math.Max(bgL*0.75, 0.03))
	}

	return &starshipColors{
		S1:   s1.hex(),
		S1FG: pickText(s1, s1, 7.0, false).hex(),
		S2:   accent.hex(),
		S2FG: pickText(accent, accent, 4.5, false).hex(),
		S3:   s3.hex(),
		S3FG: pickText(s3, accent, 3.5, true).hex(),
		S4:   s4.hex(),
		S4FG: pickText(s4, accent, 3.5, true).hex(),
		S5:   s5.hex(),
		S5FG: pickText(s5, accent, 3.5, true).hex(),
	}, nil
}

func ApplyStarshipTheme(theme *pkg.Theme) error {
	colors, err := buildStarshipColors(theme)
	if err != nil {
		return err
	}

	tmpl, err := template.New("starship").Parse(starshipTemplate)
	if err != nil {
		return fmt.Errorf("failed to parse starship template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, colors); err != nil {
		return fmt.Errorf("failed to render starship template: %w", err)
	}

	tmpDir := "/tmp/matheme"
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp dir: %w", err)
	}

	filePath := filepath.Join(tmpDir, "starship_theme.toml")
	if err := os.WriteFile(filePath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write starship theme file: %w", err)
	}

	return nil
}
