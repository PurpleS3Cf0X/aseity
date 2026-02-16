package memory

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Ensure AutoMemory implements Store
var _ Store = (*AutoMemory)(nil)

// Learning represents a single unit of persistent memory
type Learning struct {
	ID        string    `json:"id"`
	Category  string    `json:"category"` // "preference", "fact", "correction"
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
	Source    string    `json:"source"` // "user", "auto"
}

// AutoMemory manages persistent learnings across sessions
type AutoMemory struct {
	mu        sync.RWMutex
	Learnings []Learning `json:"learnings"`
	baseDir   string
}

// NewAutoMemory initializes the persistent memory
func NewAutoMemory() *AutoMemory {
	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".config", "aseity", "memory")

	m := &AutoMemory{
		baseDir:   baseDir,
		Learnings: []Learning{},
	}

	// Try to load existing
	_ = m.Load()
	return m
}

// Load reads memory from disk
func (m *AutoMemory) Load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	path := filepath.Join(m.baseDir, "auto_memory.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	return json.Unmarshal(data, &m.Learnings)
}

// Save writes memory to disk
func (m *AutoMemory) Save() error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if err := os.MkdirAll(m.baseDir, 0755); err != nil {
		return err
	}

	path := filepath.Join(m.baseDir, "auto_memory.json")
	data, err := json.MarshalIndent(m.Learnings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

// AddLearning records a new insight
func (m *AutoMemory) AddLearning(category, content string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Dedup simple exact matches
	for _, l := range m.Learnings {
		if l.Content == content && l.Category == category {
			return nil
		}
	}

	m.Learnings = append(m.Learnings, Learning{
		ID:        fmt.Sprintf("mem_%d", time.Now().UnixNano()),
		Category:  category,
		Content:   content,
		CreatedAt: time.Now(),
		Source:    "auto",
	})
	return nil
}

// RetrieveContext returns recent learnings formatted for the LLM prompt.
// This implements the Store interface.
func (m *AutoMemory) RetrieveContext() (string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if len(m.Learnings) == 0 {
		return "", nil
	}

	// Sort by newest first
	sorted := make([]Learning, len(m.Learnings))
	copy(sorted, m.Learnings)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CreatedAt.After(sorted[j].CreatedAt)
	})

	var b strings.Builder
	b.WriteString("## 🧠 Memories & Learnings\n")
	b.WriteString("Use these past learnings to guide your actions:\n\n")

	// Limit to last 10 to save tokens (naive approach, RAG comes later)
	limit := 10
	if len(sorted) < limit {
		limit = len(sorted)
	}

	for _, l := range sorted[:limit] {
		b.WriteString(fmt.Sprintf("- [%s] %s\n", strings.ToUpper(l.Category), l.Content))
	}
	b.WriteString("\n")

	return b.String(), nil
}
