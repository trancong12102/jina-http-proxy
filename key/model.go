package key

type Key struct {
	Key string
}

type InsertKeyParams struct {
	Key string
}

type KeyStats struct {
	Count   int
	Balance int64
}
