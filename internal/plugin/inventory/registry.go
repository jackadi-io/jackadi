package inventory

import (
	"fmt"
	"sync"

	"github.com/jackadi-io/jackadi/internal/plugin/core"
)

var Registry = New()

type registry struct {
	collection map[string]core.Collection
	lock       *sync.Mutex
}

func New() registry {
	return registry{
		collection: make(map[string]core.Collection),
		lock:       &sync.Mutex{},
	}
}

func (r *registry) Get(name string) (core.Collection, error) {
	r.lock.Lock()
	defer r.lock.Unlock()
	if m, ok := r.collection[name]; ok {
		return m, nil
	}
	return nil, fmt.Errorf("'%s' not registered", name)
}

func (r *registry) Register(m core.Collection) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	name, err := m.Name()
	if err != nil {
		return err
	}
	if _, exists := r.collection[name]; exists {
		return fmt.Errorf("%s already exists", name)
	}

	r.collection[name] = m
	return nil
}

func (r *registry) Unregister(name string) error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if _, exists := r.collection[name]; !exists {
		return fmt.Errorf("%s does not exist", name)
	}

	delete(r.collection, name)
	return nil
}

func (r *registry) Names() []string {
	r.lock.Lock()
	defer r.lock.Unlock()

	tasks := []string{}
	for name := range r.collection {
		tasks = append(tasks, name)
	}

	return tasks
}
