package prql

type SourceLocationError struct {
	StartLine   int
	StartColumn int
	EndLine     int
	EndColumn   int
}

type CompileMessage struct {
	ErrorCode rune
	// Annoted code containing the error and the hints
	Display string
	// The location of the error
	LocationError SourceLocationError
}
