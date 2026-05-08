package tui

type EditExpenseOption func(*editExpenseConfig)

type editExpenseConfig struct {
	mock bool
}

func WithMock(mock bool) EditExpenseOption {
	return func(c *editExpenseConfig) {
		c.mock = mock
	}
}
