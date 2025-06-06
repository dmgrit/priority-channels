package strategies

type PrioritizationStrategy[W any] interface {
	Initialize(weights []W) error
	InitializeCopy() PrioritizationStrategy[W]
	NextSelectCasesRankedIndexes(upto int) ([]RankedIndex, bool)
	UpdateOnCaseSelected(index int)
	DisableSelectCase(index int)
	EnableSelectCase(index int)
}

type RankedIndex struct {
	Index int
	Rank  int
}
