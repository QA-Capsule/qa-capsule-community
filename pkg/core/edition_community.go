//go:build !enterprise

package core

type communityEdition struct{}

func (communityEdition) Active() bool { return false }

func init() {
	currentEdition = communityEdition{}
}
