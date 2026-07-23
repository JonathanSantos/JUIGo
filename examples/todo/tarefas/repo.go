package tarefas

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
)

// Repositorio persiste as tarefas — a UI depende SÓ desta interface
// (padrão repositório: trocar o arquivo por rede ou memória não toca a
// interface nem o domínio).
type Repositorio interface {
	Carregar() ([]Tarefa, error)
	Salvar([]Tarefa) error
}

// RepositorioJSON grava as tarefas num arquivo JSON.
type RepositorioJSON struct {
	Caminho string
}

// Carregar lê o arquivo; inexistente devolve lista vazia (primeira
// execução).
func (r *RepositorioJSON) Carregar() ([]Tarefa, error) {
	dados, err := os.ReadFile(r.Caminho)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var itens []Tarefa
	if err := json.Unmarshal(dados, &itens); err != nil {
		return nil, err
	}
	return itens, nil
}

// Salvar grava o arquivo inteiro (o conjunto é pequeno; simplicidade
// primeiro).
func (r *RepositorioJSON) Salvar(itens []Tarefa) error {
	dados, err := json.MarshalIndent(itens, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(r.Caminho, dados, 0o644)
}
