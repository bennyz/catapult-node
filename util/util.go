package util

import (
	uuid "github.com/satori/go.uuid"
)

func StringToUUID(str string) uuid.UUID {
	s, err := uuid.FromString(str)
	if err != nil {
		return uuid.Nil
	}

	return s
}
