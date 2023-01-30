package testdata

import (
	"fmt"
	"strings"
)

type Foo struct {
	ID    string
	Name  string
	Price float64
}

func (f *Foo) Validate() error {
	if f.ID == "" {
		return fmt.Errorf("ID can't be empty")
	}

	if strings.ContainsAny(f.ID, "!@#$%^&*()_+") {
		return fmt.Errorf("ID contains invalid characters")
	}

	return nil
}
