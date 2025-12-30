package sections

func NewDocumenterLLMSection() *LLMSectionModel {
	return NewLLMSectionWithTarget(LLMTargetDocumenter)
}
