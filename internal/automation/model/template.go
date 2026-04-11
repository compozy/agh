package model

import (
	"errors"
	"fmt"
	"strings"
	"text/template"
	"text/template/parse"
)

// ParseTriggerPromptTemplate parses a trigger prompt template with strict activation-envelope validation.
func ParseTriggerPromptTemplate(prompt string) (*template.Template, error) {
	if strings.TrimSpace(prompt) == "" {
		return nil, errors.New("trigger prompt template is required")
	}

	tmpl, err := template.New("trigger_prompt").Option("missingkey=error").Parse(prompt)
	if err != nil {
		return nil, fmt.Errorf("parse trigger prompt template: %w", err)
	}

	for _, subtemplate := range tmpl.Templates() {
		if subtemplate.Tree == nil || subtemplate.Root == nil {
			continue
		}
		if err := validateTemplateNode(subtemplate.Root); err != nil {
			return nil, err
		}
	}

	return tmpl, nil
}

// ValidateTriggerPromptTemplate validates a trigger prompt template against the normalized activation-envelope model.
func ValidateTriggerPromptTemplate(prompt string) error {
	_, err := ParseTriggerPromptTemplate(prompt)
	return err
}

func validateTemplateNode(node parse.Node) error {
	switch n := node.(type) {
	case nil:
		return nil
	case *parse.ListNode:
		if n == nil {
			return nil
		}
		for _, child := range n.Nodes {
			if err := validateTemplateNode(child); err != nil {
				return err
			}
		}
	case *parse.ActionNode:
		if n == nil {
			return nil
		}
		return validatePipeNode(n.Pipe)
	case *parse.IfNode:
		if n == nil {
			return nil
		}
		if err := validatePipeNode(n.Pipe); err != nil {
			return err
		}
		if err := validateTemplateNode(n.List); err != nil {
			return err
		}
		return validateTemplateNode(n.ElseList)
	case *parse.RangeNode:
		if n == nil {
			return nil
		}
		if err := validatePipeNode(n.Pipe); err != nil {
			return err
		}
		if err := validateTemplateNode(n.List); err != nil {
			return err
		}
		return validateTemplateNode(n.ElseList)
	case *parse.WithNode:
		if n == nil {
			return nil
		}
		if err := validatePipeNode(n.Pipe); err != nil {
			return err
		}
		if err := validateTemplateNode(n.List); err != nil {
			return err
		}
		return validateTemplateNode(n.ElseList)
	case *parse.TemplateNode:
		if n == nil {
			return nil
		}
		return validatePipeNode(n.Pipe)
	case *parse.TextNode, *parse.CommentNode, *parse.BreakNode, *parse.ContinueNode:
		return nil
	}

	return nil
}

func validatePipeNode(pipe *parse.PipeNode) error {
	if pipe == nil {
		return nil
	}
	for _, cmd := range pipe.Cmds {
		if err := validateCommandNode(cmd); err != nil {
			return err
		}
	}
	return nil
}

func validateCommandNode(cmd *parse.CommandNode) error {
	if cmd == nil {
		return nil
	}
	if len(cmd.Args) == 0 {
		return nil
	}

	if ident, ok := cmd.Args[0].(*parse.IdentifierNode); ok && ident.Ident == "index" {
		if err := validateIndexArgs(cmd.Args[1:]); err != nil {
			return err
		}
	}

	for _, arg := range cmd.Args {
		if err := validateTemplateArg(arg); err != nil {
			return err
		}
	}

	return nil
}

func validateIndexArgs(args []parse.Node) error {
	if len(args) == 0 {
		return nil
	}

	path, ok := templateFieldPath(args[0])
	if !ok || len(path) == 0 {
		return nil
	}
	if path[0] != "Data" {
		return fmt.Errorf("unsupported index target %q; only .Data is supported for dynamic lookups", dottedPath(path))
	}
	return nil
}

func validateTemplateArg(node parse.Node) error {
	switch n := node.(type) {
	case nil:
		return nil
	case *parse.FieldNode:
		return validateActivationFieldPath(n.Ident)
	case *parse.ChainNode:
		path, ok := templateFieldPath(n)
		if !ok {
			return nil
		}
		return validateActivationFieldPath(path)
	case *parse.PipeNode:
		return validatePipeNode(n)
	case *parse.CommandNode:
		return validateCommandNode(n)
	}

	return nil
}

func templateFieldPath(node parse.Node) ([]string, bool) {
	switch n := node.(type) {
	case *parse.FieldNode:
		return append([]string(nil), n.Ident...), true
	case *parse.ChainNode:
		base, ok := templateFieldPath(n.Node)
		if !ok {
			if _, ok := n.Node.(*parse.DotNode); ok {
				base = nil
			} else {
				return nil, false
			}
		}
		return append(base, n.Field...), true
	case *parse.PipeNode:
		if n == nil || len(n.Cmds) != 1 {
			return nil, false
		}
		return templateFieldPath(n.Cmds[0])
	case *parse.CommandNode:
		if n == nil || len(n.Args) != 1 {
			return nil, false
		}
		return templateFieldPath(n.Args[0])
	case *parse.DotNode:
		return nil, true
	default:
		return nil, false
	}
}

func validateActivationFieldPath(path []string) error {
	if len(path) == 0 {
		return nil
	}

	switch path[0] {
	case "Kind", "Scope", "WorkspaceID", "Source":
		if len(path) > 1 {
			return fmt.Errorf("activation envelope field %q does not have child field %q", path[0], path[1])
		}
		return nil
	case "Data":
		return nil
	default:
		return fmt.Errorf("unknown activation envelope field %q", dottedPath(path))
	}
}

func dottedPath(path []string) string {
	if len(path) == 0 {
		return "."
	}
	return "." + strings.Join(path, ".")
}
