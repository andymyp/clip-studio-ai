package repository

import "errors"

var ErrNotFound = errors.New("entity not found")
var ErrConflict = errors.New("entity already exists")
