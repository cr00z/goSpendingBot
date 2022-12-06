package cache_lru

import (
	"container/list"
	"strconv"
	"sync"
	"testing"

	"github.com/cr00z/goSpendingBot/internal/cache"
	"github.com/stretchr/testify/assert"
)

// вставка нового элемента в неполный кэш
func TestLRUCache_Add_AddNewElement_AddedOnFront(t *testing.T) {
	// Arrange
	lru := NewLRUCache("", 3)
	lru.Add("one", "1")
	lru.Add("two", [2]int{2, 2})

	// Act
	eviction := lru.Add("three", 3)

	// Assert
	assert.False(t, eviction)
	checkElement(t, lru, "front", "three", 3)
	checkElement(t, lru, "back", "one", "1")
	assert.Equal(t, lru.Len(), 3)
}

// вставка нового элемента в полный кэш
func TestLRUCache_Add_AddNewElementFullCache_AddedAndOldestRemoved(t *testing.T) {
	// Arrange
	lru := NewLRUCache("", 3)
	lru.Add("one", "1")
	lru.Add("two", [2]int{2, 2})
	lru.Add("three", 3)

	// Act
	eviction := lru.Add("four", "4")

	// Assert
	assert.True(t, eviction)
	checkElement(t, lru, "front", "four", "4")
	checkElement(t, lru, "back", "two", [2]int{2, 2})
	assert.Equal(t, lru.Len(), 3)
}

// вставка существующего элемента в неполный кэш
func TestLRUCache_Add_AddExistElement_MoveToFrontAndNewValue(t *testing.T) {
	// Arrange
	lru := NewLRUCache("", 3)
	lru.Add("one", "1")
	lru.Add("two", 2)

	// Act
	lru.Add("one", 1)

	// Assert
	checkElement(t, lru, "front", "one", 1)
	checkElement(t, lru, "back", "two", 2)
	assert.Equal(t, lru.Len(), 2)
}

// вставка существующего элемента в полный кэш
func TestLRUCache_Add_AddExistElementFullCache_MoveToFrontAndNewValue(t *testing.T) {
	// Arrange
	lru := NewLRUCache("", 3)
	lru.Add("one", "1")
	lru.Add("two", [2]float32{2., 2.})
	lru.Add("three", 3)

	// Act
	lru.Add("one", 1)

	// Assert
	checkElement(t, lru, "front", "one", 1)
	checkElement(t, lru, "back", "two", [2]float32{2., 2.})
	assert.Equal(t, lru.Len(), 3)
}

// вставка синхронно
func TestLRUCache_Add_AddElementsSync_AllElementsInCache(t *testing.T) {
	// Arrange
	lru := NewLRUCache("", 10)
	items := []Item{
		{"1", "1"},
		{"2", [2]float32{2., 2.}},
		{"3", "3"},
	}
	for i := 4; i <= 10; i++ {
		items = append(items, Item{strconv.Itoa(i), i})
	}

	// Act
	var wg sync.WaitGroup
	wg.Add(len(items))
	for _, item := range items {
		go func(item Item) {
			lru.Add(item.key, item.value)
			wg.Done()
		}(item)
	}
	wg.Wait()

	// Assert
	for i := 1; i <= 10; i++ {
		checkElement(t, lru, "random", items[i-1].key, items[i-1].value)
	}
	assert.Equal(t, lru.Len(), 10)
}

// получение существующего элемента
func TestLRUCache_Get_ExistElement_ReturnWithoutError(t *testing.T) {
	// Arrange
	lru := NewLRUCache("", 3)
	lru.Add("one", "1")
	lru.Add("two", [2]int{2, 2})
	lru.Add("three", 3)

	// Act
	value, err := lru.Get("one")

	// Assert
	assert.Equal(t, value, "1")
	assert.NoError(t, err)
	checkElement(t, lru, "front", "one", "1")
	checkElement(t, lru, "back", "two", [2]int{2, 2})
	assert.Equal(t, lru.Len(), 3)
}

// получение несуществующего элемента
func TestLRUCache_Get_MissingElement_ReturnError(t *testing.T) {
	// Arrange
	lru := NewLRUCache("", 3)
	lru.Add("one", "1")
	lru.Add("two", [2]int{2, 2})
	lru.Add("three", 3)

	// Act
	value, err := lru.Get("four")

	// Assert
	assert.Nil(t, value)
	if assert.Error(t, err) {
		assert.Equal(t, cache.ErrElementNotInCache, err)
	}
	checkElement(t, lru, "front", "three", 3)
	checkElement(t, lru, "back", "one", "1")
	assert.Equal(t, lru.Len(), 3)
}

// удаление существующего элемента
func TestLRUCache_Delete_ExistElement_RemovedAndReturnWithoutError(t *testing.T) {
	// Arrange
	lru := NewLRUCache("", 3)
	lru.Add("one", "1")
	lru.Add("two", [2]int{2, 2})
	lru.Add("three", 3)

	// Act
	err := lru.Delete("one")

	// Assert
	assert.NoError(t, err)
	checkElement(t, lru, "front", "three", 3)
	checkElement(t, lru, "back", "two", [2]int{2, 2})
	assert.Equal(t, lru.Len(), 2)
}

// удаление несуществующего элемента
func TestLRUCache_Delete_MissingElement_ReturnError(t *testing.T) {
	// Arrange
	lru := NewLRUCache("", 3)
	lru.Add("one", "1")
	lru.Add("two", [2]int{2, 2})
	lru.Add("three", 3)

	// Act
	err := lru.Delete("four")

	// Assert
	if assert.Error(t, err) {
		assert.Equal(t, cache.ErrElementNotInCache, err)
	}
	checkElement(t, lru, "front", "three", 3)
	checkElement(t, lru, "back", "one", "1")
	assert.Equal(t, lru.Len(), 3)
}

// удаление синхронно
func TestLRUCache_Delete_DeleteElementsSync_NoExtraElementsInCache(t *testing.T) {
	// Arrange
	lru := NewLRUCache("", 10)

	items := []Item{
		{"1", "1"},
		{"2", [2]float32{2., 2.}},
		{"3", "3"},
	}
	for i := 4; i <= 10; i++ {
		items = append(items, Item{strconv.Itoa(i), i})
	}

	for i := 1; i <= 10; i++ {
		lru.Add(items[i-1].key, items[i-1].value)
	}

	// Act
	var wg sync.WaitGroup
	wg.Add(len(items) - 2)
	for i := 1; i <= len(items)-2; i++ {
		go func(item Item) {
			_ = lru.Delete(item.key)
			wg.Done()
		}(items[i-1])
	}
	wg.Wait()

	// Assert
	checkElement(t, lru, "random", "9", 9)
	checkElement(t, lru, "random", "10", 10)
	assert.Equal(t, lru.Len(), 2)
}

// helpers

func checkElement(t testing.TB,
	lru *LRUCache, side string, key string, expected interface{}) {

	t.Helper()
	var queueElem *list.Element
	actual, inMap := lru.values[key]
	assert.True(t, inMap)
	if side == "front" {
		queueElem = lru.queue.Front()
	} else if side == "back" {
		queueElem = lru.queue.Back()
	} else {
		for queueElem = lru.queue.Front(); queueElem != nil; queueElem = queueElem.Next() {
			if queueElem.Value.(*Item).key == key {
				break
			}
		}
		assert.NotNil(t, queueElem)
	}
	// value in map == value in queue
	assert.Equal(t, queueElem.Value.(*Item).value, actual.Value.(*Item).value)
	// value in map == expected
	assert.Equal(t, expected, actual.Value.(*Item).value)
}
