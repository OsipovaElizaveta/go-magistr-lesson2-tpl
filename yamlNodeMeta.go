package main

import "gopkg.in/yaml.v3"

type YamlNodeMeta struct {
	Tag             string
	TypeName        string
	IsRequired      bool
	Children        map[string]YamlNodeMeta
	IsSequence      bool
	AdditionalCheck func(yaml.Node, yaml.Node, Logger)
}

func GetStringNodeMeta() YamlNodeMeta {
	return YamlNodeMeta{Tag: "!!str", TypeName: "string"}
}

func GetIntNodeMeta() YamlNodeMeta {
	return YamlNodeMeta{Tag: "!!int", TypeName: "integer"}
}

func GetObjectNodeMeta(typeName string, children map[string]YamlNodeMeta) YamlNodeMeta {
	return YamlNodeMeta{Tag: "!!map", TypeName: typeName, Children: children}
}

func GetSeqNodeMeta(typeName string, children map[string]YamlNodeMeta) YamlNodeMeta {
	return YamlNodeMeta{Tag: "!!seq", TypeName: typeName, Children: children, IsSequence: true}
}

func (meta YamlNodeMeta) Required() YamlNodeMeta {
	meta.IsRequired = true
	return meta
}

func (meta YamlNodeMeta) WithAdditionalCheck(check func(yaml.Node, yaml.Node, Logger)) YamlNodeMeta {
	meta.AdditionalCheck = check
	return meta
}

func (meta YamlNodeMeta) ReplaceTypeName(typeName string) YamlNodeMeta {
	meta.TypeName = typeName
	return meta
}
