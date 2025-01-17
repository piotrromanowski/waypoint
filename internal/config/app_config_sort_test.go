package config

import (
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/stretchr/testify/require"
)

func TestSortVars(t *testing.T) {
	qm := func(str string) hcl.Expression {
		expr, diags := hclsyntax.ParseExpression([]byte(str), "frag.hcl", hcl.Pos{})
		if diags.HasErrors() {
			panic(diags.Error())
		}

		return expr
	}

	t.Run("sorts referenced variables before referencers", func(t *testing.T) {
		var c genericConfig
		c.EnvRaw = qm(`{
			a = "hello, ${config.env.c}"
			b = "${config.env.a}, sir"
			c = "goodbye"
		}`)
		c.InternalRaw = &hclsyntax.AnonSymbolExpr{}

		var ctx hcl.EvalContext

		pairs, err := c.sortVars(&ctx)
		require.NoError(t, err)

		require.Len(t, pairs, 3)

		require.Equal(t, pairs[0].Name, "c")
		require.Equal(t, pairs[1].Name, "a")
		require.Equal(t, pairs[2].Name, "b")
	})

	t.Run("tracks references between env and internal", func(t *testing.T) {
		var c genericConfig
		c.EnvRaw = qm(`{
			a = "hello, ${config.env.c}"
			c = "goodbye"
			d = "x: ${config.internal.b}"
		}`)

		c.InternalRaw = qm(`{
			b = "${config.env.a}, sir"
		}`)

		var ctx hcl.EvalContext

		pairs, err := c.sortVars(&ctx)
		require.NoError(t, err)

		require.Len(t, pairs, 4)

		require.Equal(t, pairs[0].Name, "c")
		require.Equal(t, pairs[1].Name, "a")
		require.Equal(t, pairs[2].Name, "b")
		require.Equal(t, pairs[3].Name, "d")
	})

	t.Run("tracks references between function calls", func(t *testing.T) {
		var c genericConfig
		c.EnvRaw = qm(`{
			a = "hello, ${upper(config.env.c)}"
			c = "goodbye"
			d = "x: ${lower(config.internal.b)}"
		}`)

		c.InternalRaw = qm(`{
			b = "${upcase(config.env.a)}, sir"
		}`)

		var ctx hcl.EvalContext

		pairs, err := c.sortVars(&ctx)
		require.NoError(t, err)

		require.Len(t, pairs, 4)

		require.Equal(t, pairs[0].Name, "c")
		require.Equal(t, pairs[1].Name, "a")
		require.Equal(t, pairs[2].Name, "b")
		require.Equal(t, pairs[3].Name, "d")
	})

	t.Run("detects mutual loops", func(t *testing.T) {
		var c genericConfig
		c.EnvRaw = qm(`{
			a = "hello, ${config.env.d}"
			c = "goodbye"
			d = "x: ${config.internal.b}"
		}`)

		c.InternalRaw = qm(`{
			b = "${config.env.a}, sir"
		}`)

		var ctx hcl.EvalContext

		_, err := c.sortVars(&ctx)
		require.Error(t, err)

		lve := err.(*VariableLoopError)

		require.Equal(t, []string{"config.env.a", "config.env.d", "config.internal.b"}, lve.LoopVars)
	})
}
