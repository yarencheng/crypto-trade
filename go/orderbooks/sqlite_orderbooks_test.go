package orderbooks

import (
	"context"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
)

func Test_SqliteOrderBooks_Start(t *testing.T) {
	t.Parallel()

	// arrange
	books, err := NewSqliteOrderBooks()
	require.NoError(t, err)

	// action
	err = books.Start(context.Background())
	require.NoError(t, err)
	defer books.Stop(context.Background())
}

func Test_SqliteOrderBooks_Stopt(t *testing.T) {
	t.Parallel()

	// arrange
	books, err := NewSqliteOrderBooks()
	require.NoError(t, err)

	// action
	err = books.Start(context.Background())
	require.NoError(t, err)

	err = books.Stop(context.Background())
	require.NoError(t, err)
}
