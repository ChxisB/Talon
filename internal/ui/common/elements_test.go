package common

import (
	"strings"
	"testing"

	"github.com/ChxisB/talon/deps/util/ansi"
	"github.com/ChxisB/talon/internal/ui/styles"
	"github.com/stretchr/testify/require"
)

func TestFormatTokensAndCostPrefixesEstimatedUsage(t *testing.T) {
	t.Parallel()

	sty := styles.DefaultPantera()

	ctx := &ModelContextInfo{
		ContextUsed:      120,
		ModelContext:     1000,
		Cost:             0,
		EstimatedUsage:   true,
	}
	rendered := formatTokensAndCost(&sty, ctx)
	actual := ansi.Strip(rendered)

	require.Contains(t, actual, "~12%")
	require.Contains(t, actual, "(120)")
	require.Contains(t, actual, "$0.00")
	require.True(t, strings.Contains(rendered, sty.ModelInfo.TokenPercentage.Render("~12%")))
}

func TestFormatTokensAndCostOmitsEstimatedPrefix(t *testing.T) {
	t.Parallel()

	sty := styles.DefaultPantera()

	ctx := &ModelContextInfo{
		ContextUsed:      120,
		ModelContext:     1000,
		Cost:             0,
		EstimatedUsage:   false,
	}
	actual := ansi.Strip(formatTokensAndCost(&sty, ctx))

	require.Contains(t, actual, "12%")
	require.NotContains(t, actual, "~12%")
}
