package object_templates

import "embed"

//go:embed .gitignore.tmpl
var GitIgnore string

//go:embed apimachinery/*
var ApimachineryRoot embed.FS

//go:embed object_kind.gotmpl
var ObjectKindTemplate string

//go:embed group_version.gotmpl
var GroupVersionTemplate string
