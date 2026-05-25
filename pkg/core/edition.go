package core

// Edition describes optional commercial capabilities compiled into the binary.
type Edition interface {
	// Active reports whether enterprise licensing is valid (always false in community builds).
	Active() bool
}

// currentEdition is set by edition_* build-tagged files.
var currentEdition Edition

// EditionActive returns whether the enterprise edition is licensed and active.
func EditionActive() bool {
	if currentEdition == nil {
		return false
	}
	return currentEdition.Active()
}
