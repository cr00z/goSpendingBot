package cache

import "errors"

var (
	ErrElementNotInCache = errors.New("element not in cache")
)

type Storager interface {
	Name() string
	Add(key string, value interface{}) bool
	Get(key string) (interface{}, error)
	Len() int
	Delete(key string) error
}
