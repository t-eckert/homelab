package names

import (
	"math/rand"
	"time"
)

var adjectives = []string{
	"brave", "clever", "gentle", "happy", "kind",
	"lively", "nice", "proud", "calm", "eager",
	"bright", "swift", "bold", "wise", "fair",
	"jolly", "keen", "wild", "quiet", "grand",
	"mighty", "noble", "quick", "sharp", "warm",
}

var nouns = []string{
	"dolphin", "eagle", "fox", "hawk", "lion",
	"otter", "panda", "raven", "tiger", "wolf",
	"bear", "deer", "falcon", "moose", "owl",
	"rabbit", "salmon", "sparrow", "whale", "zebra",
	"badger", "coyote", "ferret", "lynx", "orca",
}

func Generate() string {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	adjective := adjectives[rng.Intn(len(adjectives))]
	noun := nouns[rng.Intn(len(nouns))]
	return adjective + "-" + noun
}
