package contatos

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
)

// Repositorio persiste os contatos — a UI depende SÓ desta interface
// (padrão repositório, como no exemplo TodoMVC).
type Repositorio interface {
	Carregar() ([]Contato, error)
	Salvar([]Contato) error
}

// RepositorioJSON grava os contatos num arquivo JSON.
type RepositorioJSON struct {
	Caminho string
}

// Carregar lê o arquivo; inexistente devolve agenda vazia (primeira
// execução).
func (r *RepositorioJSON) Carregar() ([]Contato, error) {
	dados, err := os.ReadFile(r.Caminho)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var itens []Contato
	if err := json.Unmarshal(dados, &itens); err != nil {
		return nil, err
	}
	return itens, nil
}

// Salvar grava o arquivo inteiro (o conjunto é pequeno; simplicidade
// primeiro).
func (r *RepositorioJSON) Salvar(itens []Contato) error {
	dados, err := json.MarshalIndent(itens, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.Caminho, dados, 0o644)
}
