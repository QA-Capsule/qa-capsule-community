//go:build !enterprise

package core

type communityEdition struct{}

func (communityEdition) ID() string    { return "community" }
func (communityEdition) Active() bool { return false }

func init() {
	currentEdition = communityEdition{}
}
