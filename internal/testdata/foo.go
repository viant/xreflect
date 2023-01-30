package testdata

import "fmt"

type Foo struct {
	ID    string
	Name  string
	Price float64
}

func (f *Foo) Validate() error {
	if f.ID == "" {
		return fmt.Errorf("ID can't be empty")
	}

	return nil
}
