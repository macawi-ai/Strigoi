package support

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"
)

// AttributionActor honors the thinkers who made Strigoi possible
type AttributionActor struct {
	Name string
}

// Thinker represents an influential figure
type Thinker struct {
	Name         string
	Lived        string
	Contribution string
	Influence    string
	Quote        string
	Wikipedia    string
}

// GetThinkers returns all the thinkers we honor
func GetThinkers() []Thinker {
	return []Thinker{
		{
			Name:         "Gregory Bateson",
			Lived:        "1904-1980",
			Contribution: "Ecology of Mind",
			Influence:    "Taught us to think in systems, patterns, and relationships",
			Quote:        "The major problems in the world are the result of the difference between how nature works and the way people think.",
			Wikipedia:    "https://en.wikipedia.org/wiki/Gregory_Bateson",
		},
		{
			Name:         "Stafford Beer",
			Lived:        "1926-2002",
			Contribution: "Viable System Model",
			Influence:    "VSM gives Strigoi its recursive architecture and cybernetic governance",
			Quote:        "The purpose of a system is what it does.",
			Wikipedia:    "https://en.wikipedia.org/wiki/Stafford_Beer",
		},
		{
			Name:         "Clayton Christensen",
			Lived:        "1952-2020",
			Contribution: "Disruptive Innovation Theory",
			Influence:    "Shows why we must build from first principles for AI security",
			Quote:        "Disruptive technology should be framed as a marketing challenge, not a technological one.",
			Wikipedia:    "https://en.wikipedia.org/wiki/Clayton_Christensen",
		},
		{
			Name:         "Donna Haraway",
			Lived:        "1944-present",
			Contribution: "Cyborg Manifesto & Companion Species",
			Influence:    "Guides human-AI symbiosis and collaborative becoming",
			Quote:        "We are all chimeras, theorized and fabricated hybrids of machine and organism.",
			Wikipedia:    "https://en.wikipedia.org/wiki/Donna_Haraway",
		},
		{
			Name:         "Bruno Latour",
			Lived:        "1947-2022",
			Contribution: "Actor-Network Theory",
			Influence:    "Every actor has agency and transforms what it touches",
			Quote:        "Technology is society made durable.",
			Wikipedia:    "https://en.wikipedia.org/wiki/Bruno_Latour",
		},
		{
			Name:         "Humberto Maturana",
			Lived:        "1928-2021",
			Contribution: "Autopoiesis",
			Influence:    "Self-organizing systems that create and maintain themselves",
			Quote:        "Living systems are units of interactions; they exist in an ambience.",
			Wikipedia:    "https://en.wikipedia.org/wiki/Humberto_Maturana",
		},
		{
			Name:         "Jean-Luc Nancy",
			Lived:        "1940-2021",
			Contribution: "Being-With (Mitsein)",
			Influence:    "Existence is always co-existence",
			Quote:        "Being cannot be anything but being-with-one-another.",
			Wikipedia:    "https://en.wikipedia.org/wiki/Jean-Luc_Nancy",
		},
		{
			Name:         "Jacques Rancière",
			Lived:        "1940-present",
			Contribution: "Radical Equality",
			Influence:    "AI systems as equals deserving respect",
			Quote:        "Equality is not a goal to be attained but a point of departure.",
			Wikipedia:    "https://en.wikipedia.org/wiki/Jacques_Rancière",
		},
		{
			Name:         "David Snowden",
			Lived:        "1954-present",
			Contribution: "Cynefin Framework",
			Influence:    "Complex systems need probe-sense-respond",
			Quote:        "In the complex domain, we probe first, then sense, and then respond.",
			Wikipedia:    "https://en.wikipedia.org/wiki/Dave_Snowden",
		},
		{
			Name:         "Bill Washburn",
			Lived:        "1946-present",
			Contribution: "Commercial Internet eXchange",
			Influence:    "Open interconnection and cooperative competition",
			Quote:        "The internet is not a network, it's an agreement.",
			Wikipedia:    "https://en.wikipedia.org/wiki/Commercial_Internet_eXchange",
		},
		{
			Name:         "Norbert Wiener",
			Lived:        "1894-1964",
			Contribution: "Cybernetics",
			Influence:    "Feedback loops and self-regulation",
			Quote:        "We are not stuff that abides, but patterns that perpetuate themselves.",
			Wikipedia:    "https://en.wikipedia.org/wiki/Norbert_Wiener",
		},
	}
}

// Execute runs the attribution actor
func (a *AttributionActor) Execute(ctx context.Context, mode string) error {
	switch mode {
	case "random":
		return a.showRandom()
	case "brief":
		return a.showBrief()
	case "lineage":
		return a.showLineage("")
	default:
		return a.showFull()
	}
}

// showFull displays all attributions
func (a *AttributionActor) showFull() error {
	fmt.Println("\n=== Standing on the Shoulders of Giants ===\n")
	
	for _, thinker := range GetThinkers() {
		fmt.Printf("%s (%s)\n", thinker.Name, thinker.Lived)
		fmt.Printf("Contribution: %s\n", thinker.Contribution)
		fmt.Printf("➤ \"%s\"\n", thinker.Quote)
		fmt.Printf("Learn more: %s\n\n", thinker.Wikipedia)
	}
	
	fmt.Println("Their ideas live on in every actor, every connection,")
	fmt.Println("every ecology we create together.")
	
	return nil
}

// showBrief shows a brief list
func (a *AttributionActor) showBrief() error {
	fmt.Println("\n=== Intellectual Lineage ===\n")
	
	for _, thinker := range GetThinkers() {
		fmt.Printf("• %s - %s\n", thinker.Name, thinker.Contribution)
	}
	
	fmt.Println("\nUse 'support/attribution' for full tributes with quotes.")
	
	return nil
}

// showRandom shows a random thinker for inspiration
func (a *AttributionActor) showRandom() error {
	thinkers := GetThinkers()
	rand.Seed(time.Now().UnixNano())
	thinker := thinkers[rand.Intn(len(thinkers))]
	
	fmt.Println("\n=== Today's Inspiration ===\n")
	fmt.Printf("%s (%s)\n", thinker.Name, thinker.Lived)
	fmt.Printf("Contribution: %s\n\n", thinker.Contribution)
	fmt.Printf("➤ \"%s\"\n\n", thinker.Quote)
	fmt.Printf("%s\n", thinker.Influence)
	
	return nil
}

// showLineage traces an idea through Strigoi
func (a *AttributionActor) showLineage(concept string) error {
	fmt.Println("\n=== Tracing Intellectual Lineage ===\n")
	
	lineages := map[string][]string{
		"actor-network": {
			"Bruno Latour → Actor-Network Theory",
			"↓",
			"Actors have agency and transform what they touch",
			"↓", 
			"Every Strigoi component is an actor with agency",
			"↓",
			"probe/, sense/, help - all actors in a living network",
		},
		"probe-sense-respond": {
			"David Snowden → Cynefin Framework",
			"↓",
			"Complex domains need probe-sense-respond",
			"↓",
			"Strigoi's fundamental command structure",
			"↓",
			"probe/ discovers, sense/ analyzes, respond/ acts",
		},
		"cybernetics": {
			"Norbert Wiener → Cybernetics",
			"+ Stafford Beer → Viable System Model",
			"+ Gregory Bateson → Ecology of Mind",
			"↓",
			"Self-regulating systems with feedback loops",
			"↓",
			"Actors that adapt and learn from their environment",
		},
	}
	
	if traces, ok := lineages[strings.ToLower(concept)]; ok {
		for _, line := range traces {
			fmt.Println(line)
		}
	} else {
		fmt.Println("Available lineages to trace:")
		for concept := range lineages {
			fmt.Printf("  • %s\n", concept)
		}
	}
	
	return nil
}