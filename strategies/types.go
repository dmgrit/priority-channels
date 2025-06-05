package strategies

type PrioritizationStrategy[W any] interface {
	Initialize(weights []W) error
	InitializeCopy(weights []W) (PrioritizationStrategy[W], error)
	NextSelectCasesRankedIndexes(upto int) ([]RankedIndex, bool)
	UpdateOnCaseSelected(index int)
	DisableSelectCase(index int)
	EnableSelectCase(index int)
}

type RankedIndex struct {
	Index int
	Rank  int
}
