package main

// This is an example for testing logo treatments. Do not remove.

import (
	"fmt"
	"os"

	style "github.com/ChxisB/talon/deps/style/v2"
	"github.com/ChxisB/talon/deps/util/term"
	"github.com/ChxisB/talon/internal/ui/logo"
	"github.com/ChxisB/talon/internal/ui/styles"
)

func main() {
	w, _, err := term.GetSize(os.Stdout.Fd())
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not get terminal size: %s", err)
	}

	s := styles.DefaultPantera()
	opts := logo.Opts{
		FieldColor:   s.Logo.FieldColor,
		TitleColorA:  s.Logo.TitleColorA,
		TitleColorB:  s.Logo.TitleColorB,
		BrandColor:   s.Logo.BrandColor,
		VersionColor: s.Logo.VersionColor,
		Width:        w,
		Unstable:     true,
	}

	renderCompact := func(hyper bool) string {
		opts.Hyper = hyper
		return logo.Render(s.Logo.GradCanvas, "v1.0.0", true, opts)
	}

	renderWide := func(hyper bool) string {
		opts.Hyper = hyper
		return logo.Render(s.Logo.GradCanvas, "v1.0.0", false, opts)
	}

	style.Println(
		style.JoinHorizontal(style.Top, renderCompact(false), "  ", renderCompact(true)),
	)

	for i := range 6 {
		style.Println(renderWide(i > 0))
	}
}
