package storage

import "fmt"

type URLStore interface {
	Save(id string, originalURL string) error
	Get(id string) (string, error)
	Exists(id string) (bool, error)
}

type ErrNotFound struct {
	ID string
}

type ErrIDExist struct {
	ID string
}

func (e *ErrNotFound) Error() string {
	return fmt.Sprintf("ID '%s'에 해당하는 URL을 찾을 수 없습니다.", e.ID)
}

func (e *ErrIDExist) Error() string {
	return fmt.Sprintf("ID '%s'가 이미 존재합니다.")
}
