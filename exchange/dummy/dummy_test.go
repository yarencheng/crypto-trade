package dummy

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/yarencheng/gospring/application_context"
	"github.com/yarencheng/gospring/v1"
)

func TestSss(t *testing.T) {
	ctx := application_context.Default()

	err := ctx.AddConfigs(&v1.Bean{
		ID:         "dummy_exchange",
		Type:       v1.T(Dummy{}),
		FactoryFn:  New,
		Properties: []v1.Property{},
	})

	require.NoError(t, err)

	_, err = ctx.GetByID("dummy_exchange")
	require.NoError(t, err)
}
