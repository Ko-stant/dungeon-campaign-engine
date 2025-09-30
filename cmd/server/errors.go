package main

import "fmt"

// GameError represents a game logic error
type GameError struct {
	Code    string
	Message string
}

func (e *GameError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}