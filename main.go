package main

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

func main() {

	sb := mainImpl()

	if sb.Len() > 0 {
		fmt.Print(sb.String())
	}
}

func mainImpl() (sb strings.Builder) {

	if len(os.Args) < 2 {
		sb.WriteString("No filename provided\n")
		return
	}

	filePath := os.Args[1]

	logger := Logger{&sb, filePath}

	if !fileExists(filePath) {
		logger.Writeln(0, "File not found")
		return
	}

	content, err := os.ReadFile(filePath)

	if err != nil {
		logger.Writeln(0, "Cannot read file content: %v", err)
		return
	}

	var root yaml.Node
	if err := yaml.Unmarshal(content, &root); err != nil {
		logger.Writeln(0, "Cannot unmarshal file content: %v", err)
		return
	}

	if len(root.Content) != 1 {
		logger.Writeln(0, "More than one root element found")
	}

	getStringNodeMeta := func() YamlNodeMeta { return GetStringNodeMeta(checkEmpty) }

	checkContainerName := getCheckContainerName()
	checkImageName := getCheckImageName()
	checkMemory := getCheckMemory()

	probeMeta := GetObjectNodeMeta("Probe", map[string]YamlNodeMeta{
		"httpGet": GetObjectNodeMeta("HTTPGetAction", map[string]YamlNodeMeta{
			"path": getStringNodeMeta().Required().WithAdditionalCheck(checkRelativePath),
			"port": GetIntNodeMeta().Required().WithAdditionalCheck(checkPort),
		}).Required(),
	})

	resourceRequirementMeta := GetObjectNodeMeta("ResourceRequirement", map[string]YamlNodeMeta{
		"cpu":    GetIntNodeMeta(),
		"memory": getStringNodeMeta().WithAdditionalCheck(checkMemory),
	})

	visitNode(*root.Content[0], *root.Content[0], GetObjectNodeMeta("root", map[string]YamlNodeMeta{
		"apiVersion": getStringNodeMeta().Required(),
		"kind":       getStringNodeMeta().Required(),
		"metadata": GetObjectNodeMeta("ObjectMeta", map[string]YamlNodeMeta{
			"name":      getStringNodeMeta().Required(),
			"namespace": getStringNodeMeta(),
			"labels":    GetObjectNodeMeta("Object", map[string]YamlNodeMeta{}).WithAdditionalCheck(checkPlainObject),
		}).Required(),
		"spec": GetObjectNodeMeta("PodSpec", map[string]YamlNodeMeta{
			"os": getStringNodeMeta().ReplaceTypeName("PodOS").WithAdditionalCheck(checkOS),
			"containers": GetSeqNodeMeta("Container[]", map[string]YamlNodeMeta{
				"containers": GetObjectNodeMeta("Container", map[string]YamlNodeMeta{
					"name":  getStringNodeMeta().Required().WithAdditionalCheck(checkContainerName),
					"image": getStringNodeMeta().Required().WithAdditionalCheck(checkImageName),
					"ports": GetSeqNodeMeta("ContainerPort[]", map[string]YamlNodeMeta{
						"ports": GetObjectNodeMeta("ContainerPort", map[string]YamlNodeMeta{
							"containerPort": GetIntNodeMeta().Required().WithAdditionalCheck(checkPort),
							"protocol":      getStringNodeMeta().WithAdditionalCheck(checkProtocol),
						}),
					}),
					"readinessProbe": probeMeta,
					"livenessProbe":  probeMeta,
					"resources": GetObjectNodeMeta("ResourceRequirements", map[string]YamlNodeMeta{
						"requests": resourceRequirementMeta,
						"limits":   resourceRequirementMeta,
					}).Required(),
				}),
			}).Required(),
		}).Required(),
	}), logger)

	return
}

func fileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !errors.Is(err, os.ErrNotExist)
}

func visitNode(nameNode yaml.Node, valueNode yaml.Node, meta YamlNodeMeta, logger Logger) {
	requiredFields := CreateSet[string]()

	if valueNode.Tag != meta.Tag {
		logger.Writeln(nameNode.Line, "%v must be type %v", nameNode.Value, meta.TypeName)
		return
	}

	for k, v := range meta.Children {
		if v.IsRequired {
			requiredFields.Add(k)
		}
	}

	for _, check := range meta.AdditionalCheck {
		check(nameNode, valueNode, logger)
	}

	upperBoundEx := len(valueNode.Content)
	increment := 1

	if !meta.IsSequence {
		upperBoundEx--
		increment++
	}

	for i := 0; i < upperBoundEx; i += increment {
		var childNameNode yaml.Node
		if meta.IsSequence {
			childNameNode = nameNode
		} else {
			childNameNode = *valueNode.Content[i]
		}
		if fieldMeta, ok := meta.Children[childNameNode.Value]; ok {
			requiredFields.Remove(childNameNode.Value)
			visitNode(childNameNode, *valueNode.Content[i+increment-1], fieldMeta, logger)
		}
	}

	if requiredFields.Count() > 0 {
		for k := range requiredFields {
			logger.Writeln(0, "%v is required", k)
		}
	}
}

func checkEmpty(nameNode yaml.Node, valueNode yaml.Node, logger Logger) {
	if strings.TrimSpace(valueNode.Value) == "" {
		logger.Writeln(nameNode.Line, "%v is required", nameNode.Value)
	}
}

func checkPlainObject(nameNode yaml.Node, valueNode yaml.Node, logger Logger) {
	meta := GetStringNodeMeta(checkEmpty)
	for i := 0; i < len(valueNode.Content)-1; i += 2 {
		childNameNode := valueNode.Content[i]

		if valueNode.Content[i+1].Tag != meta.Tag {
			logger.Writeln(childNameNode.Line, "%v must be type %v", childNameNode.Value, meta.TypeName)
		}
	}
}

func checkOS(nameNode yaml.Node, valueNode yaml.Node, logger Logger) {
	if valueNode.Value != "linux" && valueNode.Value != "windows" {
		logger.Writeln(nameNode.Line, "%v has unsupported value '%v'", nameNode.Value, valueNode.Value)
	}
}

func getCheckContainerName() func(yaml.Node, yaml.Node, Logger) {
	names := CreateSet[string]()
	regex, _ := regexp.Compile(`^[a-z]+(?:_[a-z]+)*$`)

	return func(nameNode yaml.Node, valueNode yaml.Node, logger Logger) {
		if strings.TrimSpace(valueNode.Value) == "" {
			return
		}

		if !regex.MatchString(valueNode.Value) {
			logger.Writeln(nameNode.Line, "%v has invalid format '%v'", nameNode.Value, valueNode.Value)
			return
		}

		if names.Contains(valueNode.Value) {
			logger.Writeln(nameNode.Line, "%v must be unique", nameNode.Value)
		} else {
			names.Add(valueNode.Value)
		}
	}
}

func getCheckImageName() func(yaml.Node, yaml.Node, Logger) {
	regex, _ := regexp.Compile(`^registry\.bigbrother\.io\/[a-zA-Z]+:v\d+\.\d+\.\d+$`)

	return func(nameNode yaml.Node, valueNode yaml.Node, logger Logger) {
		if !regex.MatchString(valueNode.Value) {
			logger.Writeln(nameNode.Line, "%v has invalid format '%v'", nameNode.Value, valueNode.Value)
		}
	}
}

func checkPort(nameNode yaml.Node, valueNode yaml.Node, logger Logger) {
	number, _ := strconv.ParseInt(strings.TrimSpace(valueNode.Value), 10, 64)
	if number <= 0 || number >= 65536 {
		logger.Writeln(nameNode.Line, "%v value out of range", nameNode.Value)
	}
}

func checkProtocol(nameNode yaml.Node, valueNode yaml.Node, logger Logger) {
	if valueNode.Value != "TCP" && valueNode.Value != "UDP" {
		logger.Writeln(nameNode.Line, "%v has unsupported value '%v'", nameNode.Value, valueNode.Value)
	}
}

func checkRelativePath(nameNode yaml.Node, valueNode yaml.Node, logger Logger) {
	u, err := url.Parse(valueNode.Value)
	if err != nil || u.Scheme != "" || u.Host != "" || u.Path == "" || strings.HasPrefix(valueNode.Value, "//") {
		logger.Writeln(nameNode.Line, "%v has invalid format '%v'", nameNode.Value, valueNode.Value)
	}
}

func getCheckMemory() func(yaml.Node, yaml.Node, Logger) {
	regex, _ := regexp.Compile(`^\d+(?:Ki|Mi|Gi)$`)

	return func(nameNode yaml.Node, valueNode yaml.Node, logger Logger) {
		if !regex.MatchString(valueNode.Value) {
			logger.Writeln(nameNode.Line, "%v has invalid format '%v'", nameNode.Value, valueNode.Value)
		}
	}
}
