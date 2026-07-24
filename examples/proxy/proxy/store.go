package proxy

import "sync"

// Store guarda as trocas capturadas. É a fronteira de CONCORRÊNCIA do
// domínio: o proxy grava de goroutines do servidor HTTP e a UI lê da main
// thread — tudo protegido por mutex, e a UI recebe a notificação por um
// assinante (que a liga a App.Post). Os Exchange devolvidos são imutáveis
// após adicionados.
type Store struct {
	mu    sync.Mutex
	items []*Exchange
	seq   int
	onAdd func()
}

// NewStore cria um repositório vazio.
func NewStore() *Store {
	return &Store{}
}

// OnChange registra o assinante chamado (em alguma goroutine) sempre que a
// lista muda — a UI o liga a App.Post para saltar para a main thread.
func (s *Store) OnChange(fn func()) {
	s.mu.Lock()
	s.onAdd = fn
	s.mu.Unlock()
}

// Add anexa a troca, atribui o ID sequencial e notifica. Devolve o ID.
func (s *Store) Add(e *Exchange) int {
	s.mu.Lock()
	s.seq++
	e.ID = s.seq
	s.items = append(s.items, e)
	fn := s.onAdd
	s.mu.Unlock()
	if fn != nil {
		fn()
	}
	return e.ID
}

// Snapshot devolve uma CÓPIA da lista (segura para a UI iterar sem lock).
func (s *Store) Snapshot() []*Exchange {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*Exchange, len(s.items))
	copy(out, s.items)
	return out
}

// Get devolve a troca com o ID dado.
func (s *Store) Get(id int) (*Exchange, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.items {
		if e.ID == id {
			return e, true
		}
	}
	return nil, false
}

// Count devolve o total capturado.
func (s *Store) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.items)
}

// Clear esvazia a lista e notifica.
func (s *Store) Clear() {
	s.mu.Lock()
	s.items = nil
	fn := s.onAdd
	s.mu.Unlock()
	if fn != nil {
		fn()
	}
}
