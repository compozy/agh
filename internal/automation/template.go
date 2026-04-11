package automation

import (
	modelpkg "github.com/pedronauck/agh/internal/automation/model"
	"text/template"
)

// ParseTriggerPromptTemplate parses a trigger prompt template with strict activation-envelope validation.
func ParseTriggerPromptTemplate(prompt string) (*template.Template, error) {
	return modelpkg.ParseTriggerPromptTemplate(prompt)
}

// ValidateTriggerPromptTemplate validates a trigger prompt template against the normalized activation-envelope model.
func ValidateTriggerPromptTemplate(prompt string) error {
	return modelpkg.ValidateTriggerPromptTemplate(prompt)
}
