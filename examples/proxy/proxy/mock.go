package proxy

import (
	"strings"
	"sync"
)

// MockRule intercepta chamadas que casam (método + trecho da URL) e
// responde com uma resposta SIMULADA — sem tocar no servidor real. Método
// vazio ou "*" casa qualquer um; URLContains vazio casa qualquer URL.
type MockRule struct {
	ID          int
	Enabled     bool
	Method      string
	URLContains string
	Status      int
	ContentType string
	Body        string
}

// Matches informa se a regra (habilitada) intercepta a chamada dada.
func (m *MockRule) Matches(method, url string) bool {
	if !m.Enabled {
		return false
	}
	if m.Method != "" && m.Method != "*" && !strings.EqualFold(m.Method, method) {
		return false
	}
	return m.URLContains == "" || strings.Contains(url, m.URLContains)
}

// Label descreve a regra para a lista.
func (m *MockRule) Label() string {
	met := m.Method
	if met == "" {
		met = "*"
	}
	alvo := m.URLContains
	if alvo == "" {
		alvo = "(qualquer URL)"
	}
	return met + " " + alvo
}

// Mocks é o conjunto THREAD-SAFE de regras de simulação (o proxy consulta
// de goroutines; a UI edita da main thread).
type Mocks struct {
	mu    sync.Mutex
	rules []*MockRule
	seq   int
}

// NewMocks cria um conjunto vazio.
func NewMocks() *Mocks {
	return &Mocks{}
}

// Add insere a regra, atribui o ID e a devolve.
func (m *Mocks) Add(r *MockRule) *MockRule {
	m.mu.Lock()
	m.seq++
	r.ID = m.seq
	m.rules = append(m.rules, r)
	m.mu.Unlock()
	return r
}

// Match devolve a PRIMEIRA regra habilitada que intercepta a chamada.
func (m *Mocks) Match(method, url string) (*MockRule, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rules {
		if r.Matches(method, url) {
			return r, true
		}
	}
	return nil, false
}

// Snapshot devolve uma cópia das regras (para a UI iterar).
func (m *Mocks) Snapshot() []*MockRule {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*MockRule, len(m.rules))
	copy(out, m.rules)
	return out
}

// Toggle liga/desliga a regra com o ID dado.
func (m *Mocks) Toggle(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range m.rules {
		if r.ID == id {
			r.Enabled = !r.Enabled
			return
		}
	}
}

// Remove exclui a regra com o ID dado.
func (m *Mocks) Remove(id int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i, r := range m.rules {
		if r.ID == id {
			m.rules = append(m.rules[:i], m.rules[i+1:]...)
			return
		}
	}
}

// Count devolve o total de regras.
func (m *Mocks) Count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.rules)
}
