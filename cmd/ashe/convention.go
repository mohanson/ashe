package main

import (
	"errors"
)

var (
	ErrNameHasExists = errors.New("name has exists")
	ErrNameNotExists = errors.New("name not exists")
)
