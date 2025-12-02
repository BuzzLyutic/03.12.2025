package storage

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/BuzzLyutic/03.12.2025/internal/model"
)

var (
	ErrNotFound = errors.New("link set not found")
)

// persistedData - структура для сохранения на диск
type persistedData struct {
	LinkSets     map[int64]*model.LinkSet  `json:"link_sets"`
	Counter      int64                     `json:"counter"`
	PendingTasks []model.PendingTask       `json:"pending_tasks,omitempty"`
}

type Storage struct {
	mu           sync.RWMutex
	linkSets     map[int64]*model.LinkSet
	counter      int64
	pendingTasks []model.PendingTask
	filePath     string
}

func New(filePath string) (*Storage, error) {
	// Создаём директорию если не существует
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	s := &Storage{
		linkSets: make(map[int64]*model.LinkSet),
		counter:  0,
		filePath: filePath,
	}

	// Загружаем существующие данные
	if err := s.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return s, nil
}

func (s *Storage) load() error {
	data, err := os.ReadFile(s.filePath)
	if err != nil {
		return err
	}

	var persisted persistedData
	if err := json.Unmarshal(data, &persisted); err != nil {
		return err
	}

	s.linkSets = persisted.LinkSets
	if s.linkSets == nil {
		s.linkSets = make(map[int64]*model.LinkSet)
	}
	s.counter = persisted.Counter
	s.pendingTasks = persisted.PendingTasks

	return nil
}

func (s *Storage) Save() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	persisted := persistedData{
		LinkSets:     s.linkSets,
		Counter:      s.counter,
		PendingTasks: s.pendingTasks,
	}

	data, err := json.MarshalIndent(persisted, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.filePath, data, 0644)
}

func (s *Storage) NextID() int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.counter++
	return s.counter
}

func (s *Storage) AddLinkSet(id int64, links map[string]string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.linkSets[id] = &model.LinkSet{
		ID:    id,
		Links: links,
	}
}

func (s *Storage) GetLinkSet(id int64) (*model.LinkSet, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	set, ok := s.linkSets[id]
	if !ok {
		return nil, ErrNotFound
	}
	return set, nil
}

func (s *Storage) GetLinkSets(ids []int64) []*model.LinkSet {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*model.LinkSet
	for _, id := range ids {
		if set, ok := s.linkSets[id]; ok {
			result = append(result, set)
		}
	}
	return result
}

// AddPendingTask добавляет задачу в очередь ожидания
func (s *Storage) AddPendingTask(task model.PendingTask) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pendingTasks = append(s.pendingTasks, task)
}

// GetAndClearPendingTasks возвращает и очищает pending задачи
func (s *Storage) GetAndClearPendingTasks() []model.PendingTask {
	s.mu.Lock()
	defer s.mu.Unlock()

	tasks := s.pendingTasks
	s.pendingTasks = nil
	return tasks
}

// RemovePendingTask удаляет задачу из очереди
func (s *Storage) RemovePendingTask(id int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, task := range s.pendingTasks {
		if task.ID == id {
			s.pendingTasks = append(s.pendingTasks[:i], s.pendingTasks[i+1:]...)
			return
		}
	}
}
