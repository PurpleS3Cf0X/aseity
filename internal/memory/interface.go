package memory

// Store defines the interface for persistent memory storage.
// This allows swapping the simple JSON backend for a Vector DB (RAG) in the future.
type Store interface {
	// AddLearning records a new insight or fact.
	AddLearning(category, content string) error

	// RetrieveContext returns relevant learnings formatted for the LLM prompt.
	// In the future, this can accept a query string for semantic search.
	RetrieveContext() (string, error)

	// Save persists the memory to disk (if applicable).
	Save() error
}
