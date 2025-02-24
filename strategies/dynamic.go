package strategies

type DynamicSubStrategy interface {
	InitializeWithTypeAssertion(weights []interface{}) error
	NextSelectCasesIndexes(upto int) []int
	UpdateOnCaseSelected(index int)
}

type Dynamic struct {
	strategiesByName          map[string]DynamicSubStrategy
	currentStrategyName       string
	currentStrategySelector   func() string
	origWeightsByStrategyName map[string][]interface{}
}

func NewDynamic(
	strategiesByName map[string]DynamicSubStrategy,
	currentStrategySelector func() string) *Dynamic {
	return &Dynamic{
		strategiesByName:        strategiesByName,
		currentStrategySelector: currentStrategySelector,
	}
}

func (s *Dynamic) Initialize(weights []map[string]interface{}) error {
	s.origWeightsByStrategyName = make(map[string][]interface{})

	for _, weightByStrategyName := range weights {
		for strategyName, weight := range weightByStrategyName {
			s.origWeightsByStrategyName[strategyName] = append(s.origWeightsByStrategyName[strategyName], weight)
		}
	}

	for strategyName, strategy := range s.strategiesByName {
		if err := strategy.InitializeWithTypeAssertion(s.origWeightsByStrategyName[strategyName]); err != nil {
			return err
		}
	}
	s.currentStrategyName = s.currentStrategySelector()
	return nil
}

func (s *Dynamic) NextSelectCasesIndexes(upto int) []int {
	currentStrategyName := s.currentStrategySelector()
	if currentStrategyName != s.currentStrategyName {
		s.currentStrategyName = currentStrategyName
	}
	strategy := s.strategiesByName[currentStrategyName]
	return strategy.NextSelectCasesIndexes(upto)
}

func (s *Dynamic) UpdateOnCaseSelected(index int) {
	strategy := s.strategiesByName[s.currentStrategyName]
	strategy.UpdateOnCaseSelected(index)
}
