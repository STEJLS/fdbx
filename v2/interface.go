package fdbx

import (
	"context"

	"github.com/apple/foundationdb/bindings/go/src/fdb"
	"github.com/shestakovda/errx"
)

// Key - некоторый ключ в БД, может использоваться в разных контекстах
type Key fdb.Key

// KeyWrapper - преобразователь ключа, может делать с ним любые трансформации
type KeyWrapper func(Key) (Key, error)

// ValueWrapper - преобразователь значения, может делать с ним любые трансформации
type ValueWrapper func([]byte) ([]byte, error)

// Pair - пара ключ/значение, с возможностью трансформации
type Pair interface {
	Key() (Key, error)
	Value() ([]byte, error)

	WrapKey(KeyWrapper) Pair
	WrapValue(ValueWrapper) Pair
}

// ListGetter - метод для отложенного получения списка значений
type ListGetter func() []Pair

// Waiter - объект ожидания изменения ключа
type Waiter interface {
	Resolve(context.Context) error
}

var (
	ErrKey   = errx.New("Ошибка загрузки ключа")
	ErrValue = errx.New("Ошибка загрузки значения")
)
