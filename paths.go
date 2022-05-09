package ortfomk

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/antonmedv/expr"
	exprVM "github.com/antonmedv/expr/vm"
)

// EvaluateDynamicPathExpression evaluates a path expression (that does not contain the leading ":" or the surrounding "[" and "]"
// and returns the evaluated expression, as a boolean (second return value) if the result is a boolean or as a string (first return value) if
// the result is anything else (stringifying the type with "%s"). If the result is an empty string, it becomes indistinguishable from a false boolean result.
// This is within expectations: an empty string, as well as a false boolean, means that this hydration with this path should not be rendered.
func EvaluateDynamicPathExpression(h *Hydration, expression string) (stringResult string, boolResult bool, err error) {
	LogDebug("Evaluating dynamic path expression %q", expression)
	// Expand "is" assertions syntax
	if matches := regexp.MustCompile(`^(\w+)\s+is\s+(.+)$`).FindStringSubmatch(expression); matches != nil {
		expression = fmt.Sprintf(`%s == %s ? %s : ""`, matches[1], matches[2], matches[2])
	}
	var compiledExpr *exprVM.Program
	if cached, ok := DynamicPathExpressionsCahe[expression]; ok {
		compiledExpr = cached
	} else {
		compiledExpr, err = expr.Compile(expression)
		DynamicPathExpressionsCahe[expression] = compiledExpr
	}
	if err != nil {
		return "", false, fmt.Errorf("invalid dynamic path expression %q: %w", expression, err)
	}
	value, err := expr.Run(compiledExpr, map[string]interface{}{
		"work":       h.work,
		"tag":        h.tag,
		"tech":       h.tech,
		"technology": h.tech,
		"site":       h.site,
		"language":   h.language,
		"lang":       h.language,
	})
	if err != nil {
		return "", false, fmt.Errorf("couldn't evaluate expression %q: %w", expression, err)
	}
	switch coerced := value.(type) {
	case bool:
		boolResult = coerced
	default:
		stringResult = fmt.Sprintf("%s", coerced)
	}
	return
}

// ExtractDynamicPathExpression extracts the path expression from a path.
// If the path is not a path expression, it returns an empty string.
// The extension argument is used to strip a potential extension from the path, to not let it be part of the expression when using the ":expression" syntax.
func ExtractDynamicPathExpression(path string, extension string) string {
	if strings.HasPrefix(path, ":") {
		return strings.TrimSuffix(path[1:], extension)
	} else if strings.HasPrefix(path, "[") && strings.HasSuffix(path, "]") {
		return path[1 : len(path)-1]
	} else {
		return ""
	}
}

// DynamicPathExpressions returns a list of all dynamic path expressions if the given path.
func DynamicPathExpressions(path string) (expressions []string) {
	parts := strings.Split(path, string(filepath.Separator))
	for i, part := range parts {
		extension := ""
		if i == len(parts)-1 {
			extension = filepath.Ext(part)
		}
		if expression := ExtractDynamicPathExpression(part, extension); expression != "" {
			expressions = append(expressions, expression)
		}
	}
	return
}

// EvaluateDynamicPathExpression evaluates a path that mau contain parts that are dynamic path expressions.
func EvaluateDynamicPath(h *Hydration, path string) (string, error) {
	evaluatedParts := make([]string, 0)
	parts := strings.Split(path, string(filepath.Separator))
	for i, part := range parts {
		var evaluatedPart string
		extension := ""
		if i == len(parts)-1 {
			extension = filepath.Ext(part)
		}
		if expression := ExtractDynamicPathExpression(part, extension); expression != "" {
			stringResult, boolResult, err := EvaluateDynamicPathExpression(h, expression)
			if err != nil {
				return "", fmt.Errorf("invalid dynamic path part %q: %w", part, err)
			}
			if stringResult != "" {
				evaluatedPart = stringResult
			} else if boolResult {
				evaluatedPart = part
			} else {
				return "", nil
			}
			if extension != "" {
				evaluatedPart += extension
			}
		} else {
			evaluatedPart = part
		}
		evaluatedParts = append(evaluatedParts, evaluatedPart)
	}
	return filepath.Join(evaluatedParts...), nil
}

// GetDistFilepath evaluates dynamic paths and replaces src/ with dist/.
// An empty return value means the path shouldn't be rendered with this hydration.
func (h *Hydration) GetDistFilepath(srcFilepath string) (string, error) {
	// Turn into a dist/ path
	outPath := filepath.Join("dist", GetPathRelativeToSrcDir(srcFilepath))
	outPath, err := EvaluateDynamicPath(h, outPath)
	LogDebug("after evaluation, path is %q", outPath)
	if err != nil {
		return "", fmt.Errorf("couldn't evaluate dynamic path %q: %w", outPath, err)
	}
	if strings.HasSuffix(outPath, ".pug") {
		outPath = strings.TrimSuffix(outPath, ".pug") + ".html"
	}
	// If it's a future .pdf file, remove .html/.pug to keep .pdf alone
	if regexp.MustCompile(`\.pdf\.(html|pug)$`).MatchString(outPath) {
		outPath = strings.TrimSuffix(outPath, ".pug")
		outPath = strings.TrimSuffix(outPath, ".html")
	}
	return outPath, nil
}
