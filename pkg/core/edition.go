package core

// Edition describes optional commercial capabilities compiled into the binary.
type Edition interface {
	// ID is "community" or "enterprise" (build tag).
	ID() string
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

// EditionID returns "community" or "enterprise" for UI and API discovery.
func EditionID() string {
	if currentEdition == nil {
		return "community"
	}
	return currentEdition.ID()
}
