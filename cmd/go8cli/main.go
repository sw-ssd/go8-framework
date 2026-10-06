package main

import (
	"bufio"
	"bytes"
	"embed"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"gopkg.in/yaml.v3"
)

//go:embed templates/*.tmpl
var templateFS embed.FS

const modulePath = "codeberg.org/gmhafiz/go8"

type Field struct {
	Name        string
	Camel       string
	Snake       string
	GoType      string
	ProtoType   string
	EntType     string
	Column      string
	ProtoGet    string
	ProtoAssign string
	ProtoName   string
	TsDefault    string
	Optional     bool
	ProtoNum    int
	CreateNum   int
	UpdateNum   int
}

type Data struct {
	Name         string
	Lower        string
	Plural       string
	PluralLower  string
	Fields       []Field
	HasDate      bool
	CreatedAtNum int
	UpdatedAtNum int
	DeletedAtNum int
}

// targets per subcommand
var targetsFor = map[string][]string{
	"model":             {"ent", "migration", "model", "request", "filters"},
	"create":            {"usecase", "repository", "service"},
	"read":              {"usecase", "repository", "service"},
	"update":            {"usecase", "repository", "service"},
	"delete":            {"usecase", "repository", "service"},
	"resource":          {"proto", "ent", "migration", "model", "request", "filters", "usecase", "repository", "service", "frontend"},
}

// output path per target
func outPath(target string, d *Data) string {
	switch target {
	case "proto":
		return filepath.Join("api", "go8", "v1", d.Lower+".proto")
	case "ent":
		return filepath.Join("ent", "schema", d.Lower+".go")
	case "migration":
		ts := time.Now().Format("20060102150405")
		return filepath.Join("database", "migrations", ts+"_create_"+d.PluralLower+".sql")
	case "model":
		return filepath.Join("internal", "domain", d.Lower, "model.go")
	case "request":
		return filepath.Join("internal", "domain", d.Lower, "request.go")
	case "filters":
		return filepath.Join("internal", "domain", d.Lower, "filters.go")
	case "usecase":
		return filepath.Join("internal", "domain", d.Lower, "usecase", "usecase.go")
	case "repository":
		return filepath.Join("internal", "domain", d.Lower, "repository", "postgres.go")
	case "service":
		return filepath.Join("internal", "domain", d.Lower, "service", "service.go")
	case "frontend":
		return filepath.Join("web", "src", "features", d.Lower, d.Name+"Page.tsx")
	}
	return ""
}

func main() {
	args := os.Args[1:]
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}
	if args[0] == "generate" { // optional buffalo-style prefix
		args = args[1:]
	}
	if len(args) == 0 {
		usage()
		os.Exit(1)
	}
	sub := args[0]
	switch sub {
	case "resource", "model", "create", "read", "update", "delete":
		runGenerate(sub, args[1:])
	case "run":
		runManifest(args[1:])
	default:
		// custom template: go8cli <tmplname> <ResourceName> [--field ...]
		runCustomTemplate(sub, args[1:])
	}
}

func runGenerate(sub string, args []string) {
	fs := flag.NewFlagSet(sub, flag.ExitOnError)
	name := fs.String("name", "", "resource name (PascalCase)")
	fields := &fieldList{}
	fs.Var(fields, "field", "field as name:type (repeatable), e.g. --field title:string --field priority:int")
	interactive := fs.Bool("interactive", false, "prompt for name and fields")

	rest := args
	namePos := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		namePos = args[0]
		rest = args[1:]
	}
	fs.Parse(rest)

	resName := *name
	if resName == "" {
		resName = namePos
	}
	if resName == "" && fs.NArg() > 0 {
		resName = fs.Arg(0)
	}
	resName, fieldsParsed := resolveNameAndFields(resName, *fields, *interactive, fs.Args())
	d := buildData(resName, fieldsParsed)

	for _, t := range targetsFor[sub] {
		renderTarget(t, &d)
	}
	if contains(targetsFor[sub], "service") {
		appendToDomains(&d)
	}
	fmt.Printf("generated %q (%s): %d fields\n", resName, sub, len(fieldsParsed))
	if contains(targetsFor[sub], "proto") || contains(targetsFor[sub], "ent") {
		fmt.Println("next: go generate ./ent && task gen   # regenerate ent + connect stubs (Go+TS)")
	}
}

func resolveNameAndFields(flagName string, fields fieldList, interactive bool, positional []string) (string, []Field) {
	name := flagName
	if name == "" && len(positional) > 0 && !strings.HasPrefix(positional[0], "-") {
		name = positional[0]
	}
	if interactive {
		if name == "" {
			name = prompt("Resource name (PascalCase): ")
		}
		fmt.Println("Enter fields as name:type (empty line to finish):")
		for {
			line := prompt("  field> ")
			if line == "" {
				break
			}
			f, err := parseField(line)
			if err != nil {
				fmt.Println("  invalid:", err)
				continue
			}
			fields = append(fields, f)
		}
	}
	return name, []Field(fields)
}

func buildData(name string, fields []Field) Data {
	if name == "" {
		fail("resource name is required")
	}
	lower := snake(name) // resource name is PascalCase; snake gives lower
	// for singular lower, use the snake of the singular; pluralize for table/list names
	pluralLower := pluralize(lower)
	data := Data{
		Name:        pascal(name),
		Lower:       lower,
		Plural:      pascal(pluralLower),
		PluralLower: pluralLower,
		Fields:      fields,
	}
	n := len(fields)
	for i := range fields {
		fields[i].ProtoNum = i + 2
		fields[i].CreateNum = i + 1
		fields[i].UpdateNum = i + 2
		fields[i].Camel = camel(fields[i].Name)
		if fields[i].GoType == "time.Time" {
			data.HasDate = true
		}
	}
	data.CreatedAtNum = n + 2
	data.UpdatedAtNum = n + 3
	data.DeletedAtNum = n + 4
	return data
}

func renderTarget(target string, d *Data) {
	tmplName := target + ".tmpl"
	var content []byte
	var err error
	if target == "frontend" {
		content, err = renderTemplateDelims(tmplName, d, "[[", "]]")
	} else {
		content, err = renderTemplate(tmplName, d)
	}
	if err != nil {
		fail("render %s: %v", tmplName, err)
	}
	out := outPath(target, d)
	if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
		fail("mkdir: %v", err)
	}
	if err := os.WriteFile(out, content, 0o644); err != nil {
		fail("write %s: %v", out, err)
	}
	fmt.Println("  wrote", out)
}

// renderTemplateDelims resolves a built-in (or user) template and parses it with custom delimiters.
func renderTemplateDelims(tmplName string, d *Data, left, right string) ([]byte, error) {
	for _, dir := range []string{os.Getenv("GO8_TEMPLATES"), ".go8/templates"} {
		if dir == "" {
			continue
		}
		p := filepath.Join(dir, tmplName)
		if b, err := os.ReadFile(p); err == nil {
			return renderBytesDelims(b, d, left, right)
		}
	}
	b, err := templateFS.ReadFile("templates/" + tmplName)
	if err != nil {
		return nil, err
	}
	return renderBytesDelims(b, d, left, right)
}

func renderBytesDelims(b []byte, d *Data, left, right string) ([]byte, error) {
	t, err := template.New("t").Delims(left, right).Parse(string(b))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, d); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// renderTemplate resolves a built-in template, or a user template from --templates/.go8/templates.
func renderTemplate(tmplName string, d *Data) ([]byte, error) {
	// user templates override built-in
	for _, dir := range []string{os.Getenv("GO8_TEMPLATES"), ".go8/templates"} {
		if dir == "" {
			continue
		}
		p := filepath.Join(dir, tmplName)
		if b, err := os.ReadFile(p); err == nil {
			return renderBytes(b, d)
		}
	}
	b, err := templateFS.ReadFile("templates/" + tmplName)
	if err != nil {
		return nil, err
	}
	return renderBytes(b, d)
}

func renderBytes(b []byte, d *Data) ([]byte, error) {
	t, err := template.New("t").Parse(string(b))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, d); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func appendToDomains(d *Data) {
	const path = "internal/server/domains.go"
	b, err := os.ReadFile(path)
	if err != nil {
		fail("read %s: %v", path, err)
	}
	content := string(b)
	importLine := fmt.Sprintf("\t%sService \"%s/internal/domain/%s/service\"", d.Name, modulePath, d.Lower)
	entryLine := fmt.Sprintf("\t{Name: %q, Register: %sService.Register},", d.Lower, d.Name)
	if strings.Contains(content, fmt.Sprintf("internal/domain/%s/service", d.Lower)) {
		return // already registered (idempotent)
	}
	content = strings.Replace(content, "\t// go8cli:imports", importLine+"\n\t// go8cli:imports", 1)
	content = strings.Replace(content, "\t// go8cli:domains", entryLine+"\n\t// go8cli:domains", 1)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		fail("write %s: %v", path, err)
	}
	fmt.Println("  registered", d.Lower, "in", path)
}

// --- extensibility: custom commands via manifest ---

func runManifest(args []string) {
	fs := flag.NewFlagSet("run", flag.ExitOnError)
	fs.Parse(args)
	if fs.NArg() == 0 {
		fail("usage: go8cli run <command>")
	}
	cmdName := fs.Arg(0)
	manifestPath := filepath.Join(".go8", "commands", cmdName+".yaml")
	b, err := os.ReadFile(manifestPath)
	if err != nil {
		fail("read manifest %s: %v", manifestPath, err)
	}
	var m Manifest
	if err := yaml.Unmarshal(b, &m); err != nil {
		fail("parse manifest: %v", err)
	}
	vars := map[string]string{}
	for _, p := range m.Prompts {
		val := os.Getenv(strings.ToUpper(p.Name))
		if val == "" {
			val = prompt(p.Message + ": ")
		}
		vars[p.Name] = val
	}
	ctx := struct {
		Data
		Vars map[string]string
	}{Data: buildData(m.Name, nil), Vars: vars}
	for _, step := range m.Steps {
		if step.Shell != "" {
			runShell(step.Shell, vars)
			continue
		}
		content, err := renderTemplate(step.Template+".tmpl", &ctx.Data)
		if err != nil {
			fail("render step %s: %v", step.Template, err)
		}
		out := expandVars(step.Out, vars)
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			fail("mkdir: %v", err)
		}
		if step.Mode == "append" {
			f, _ := os.ReadFile(out)
			content = append(f, content...)
		}
		if err := os.WriteFile(out, content, 0o644); err != nil {
			fail("write %s: %v", out, err)
		}
		fmt.Println("  wrote", out)
	}
}

func runCustomTemplate(tmplName string, args []string) {
	fs := flag.NewFlagSet(tmplName, flag.ExitOnError)
	name := fs.String("name", "", "resource name")
	fields := &fieldList{}
	fs.Var(fields, "field", "field as name:type")
	rest := args
	namePos := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		namePos = args[0]
		rest = args[1:]
	}
	fs.Parse(rest)
	resName := *name
	if resName == "" {
		resName = namePos
	}
	if resName == "" && fs.NArg() > 0 {
		resName = fs.Arg(0)
	}
	_, parsed := resolveNameAndFields(resName, *fields, false, fs.Args())
	d := buildData(resName, parsed)
	content, err := renderTemplate(tmplName+".tmpl", &d)
	if err != nil {
		fail("render %s: %v", tmplName, err)
	}
	out := tmplName + ".out"
	if err := os.WriteFile(out, content, 0o644); err != nil {
		fail("write: %v", err)
	}
	fmt.Println("wrote", out)
}

// --- helpers ---

type fieldList []Field

func (f *fieldList) String() string { var parts []string; for _, fl := range *f { parts = append(parts, fl.Name) }; return strings.Join(parts, ", ") }
func (f *fieldList) Set(v string) error {
	fld, err := parseField(v)
	if err != nil {
		return err
	}
	*f = append(*f, fld)
	return nil
}

func parseField(s string) (Field, error) {
	optional := false
	raw := s
	if strings.HasSuffix(raw, "?") {
		optional = true
		raw = strings.TrimSuffix(raw, "?")
	}
	parts := strings.SplitN(raw, ":", 2)
	if len(parts) != 2 {
		return Field{}, fmt.Errorf("expected name:type")
	}
	name := pascal(parts[0])
	if name == "" {
		return Field{}, fmt.Errorf("empty field name")
	}
	return fieldType(name, snake(parts[0]), parts[1], optional)
}

func fieldType(name, snakeName, typ string, optional bool) (Field, error) {
	protoName := simplePascal(snakeName)
	base := func(goType, protoType, entType, column, protoGet, protoAssign, tsDefault string) Field {
		return Field{
			Name: name, Snake: snakeName, GoType: goType, ProtoType: protoType,
			EntType: entType, Column: column, ProtoGet: protoGet, ProtoAssign: protoAssign,
			ProtoName: protoName, TsDefault: tsDefault, Optional: optional,
		}
	}
	switch typ {
	case "string":
		return base("string", "string", "String", "TEXT", "req.Msg.Get"+protoName+"()", "s."+name, `""`), nil
	case "int":
		return base("int", "int32", "Int", "INTEGER", "int(req.Msg.Get"+protoName+"())", "int32(s."+name+")", "0"), nil
	case "int64":
		return base("int64", "int64", "Int64", "BIGINT", "req.Msg.Get"+protoName+"()", "s."+name, "0"), nil
	case "bool":
		return base("bool", "bool", "Bool", "BOOLEAN", "req.Msg.Get"+protoName+"()", "s."+name, "false"), nil
	case "float":
		return base("float64", "float", "Float64", "DOUBLE PRECISION", "float64(req.Msg.Get"+protoName+"())", "float32(s."+name+")", "0"), nil
	case "date", "time", "datetime":
		return base("time.Time", "google.protobuf.Timestamp", "Time", "TIMESTAMPTZ", "req.Msg.Get"+protoName+"().AsTime()", "timestamppb.New(s."+name+")", "new Date(0)"), nil
	default:
		return Field{}, fmt.Errorf("unsupported type %q (string|int|int64|bool|float|date)", typ)
	}
}

var initialisms = map[string]string{
	"acl": "ACL", "api": "API", "ascii": "ASCII", "cpu": "CPU", "css": "CSS",
	"dns": "DNS", "eof": "EOF", "guid": "GUID", "html": "HTML", "http": "HTTP",
	"https": "HTTPS", "id": "ID", "ip": "IP", "json": "JSON", "lhs": "LHS",
	"qps": "QPS", "ram": "RAM", "rhs": "RHS", "rpc": "RPC", "sla": "SLA",
	"smtp": "SMTP", "sql": "SQL", "ssh": "SSH", "tcp": "TCP", "tls": "TLS",
	"ttl": "TTL", "udp": "UDP", "ui": "UI", "uid": "UID", "uuid": "UUID",
	"uri": "URI", "url": "URL", "utf8": "UTF8", "vm": "VM", "xml": "XML",
	"xmpp": "XMPP", "xsrf": "XSRF", "xss": "XSS",
}

func pascal(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '_' || r == '-' || r == ' ' })
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		if up, ok := initialisms[strings.ToLower(p)]; ok {
			b.WriteString(up)
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]) + p[1:])
	}
	return b.String()
}

// camel produces protobuf-es style camelCase (e.g. image_url -> imageUrl).
func camel(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '_' || r == '-' || r == ' ' })
	var b strings.Builder
	for i, p := range parts {
		if p == "" {
			continue
		}
		if i == 0 {
			b.WriteString(strings.ToLower(p))
		} else {
			b.WriteString(strings.ToUpper(p[:1]) + strings.ToLower(p[1:]))
		}
	}
	return b.String()
}
// simplePascal is protoc-gen-go style PascalCase (no initialism expansion).
func simplePascal(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == '_' || r == '-' || r == ' ' })
	var b strings.Builder
	for _, p := range parts {
		if p == "" {
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]) + p[1:])
	}
	return b.String()
}

func snake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if r >= 'A' && r <= 'Z' {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(r - 'A' + 'a')
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func pluralize(s string) string {
	if strings.HasSuffix(s, "y") && len(s) > 1 && !isVowel(rune(s[len(s)-2])) {
		return s[:len(s)-1] + "ies"
	}
	if strings.HasSuffix(s, "s") || strings.HasSuffix(s, "x") || strings.HasSuffix(s, "ch") || strings.HasSuffix(s, "sh") {
		return s + "es"
	}
	return s + "s"
}

func isVowel(r rune) bool { return r == 'a' || r == 'e' || r == 'i' || r == 'o' || r == 'u' }

func prompt(msg string) string {
	fmt.Print(msg)
	reader := bufio.NewReader(os.Stdin)
	line, _ := reader.ReadString('\n')
	return strings.TrimSpace(line)
}

func runShell(sh string, vars map[string]string) {
	expanded := expandVars(sh, vars)
	cmd := exec.Command("sh", "-c", expanded)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Run(); err != nil {
		fail("shell: %v", err)
	}
}

func expandVars(s string, vars map[string]string) string {
	for k, v := range vars {
		s = strings.ReplaceAll(s, "${"+k+"}", v)
	}
	return s
}

func contains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func fail(format string, a ...interface{}) {
	fmt.Fprintf(os.Stderr, "go8cli: "+format+"\n", a...)
	os.Exit(1)
}

func usage() {
	fmt.Println(`go8cli - generate CONNECT/ent CRUD scaffolds for go8

Usage:
  go8cli [generate] resource <Name> [--field name:type ...] [--interactive]
  go8cli [generate] model    <Name> [--field ...] [--interactive]
  go8cli [generate] create|read|update|delete <Name> [--field ...]
  go8cli <template> <Name> [--field ...]     # render a custom .tmpl
  go8cli run <command>                        # run a .go8/commands/<command>.yaml manifest

Types: string | int | int64 | bool | float`)
}

type Manifest struct {
	Name        string  `yaml:"name"`
	Description string  `yaml:"description"`
	Prompts     []Prompt `yaml:"prompts"`
	Steps       []Step  `yaml:"steps"`
}
type Prompt struct {
	Name    string `yaml:"name"`
	Message string `yaml:"message"`
	Default string `yaml:"default"`
}
type Step struct {
	Template string `yaml:"template"`
	Out      string `yaml:"out"`
	Mode     string `yaml:"mode"`
	Shell    string `yaml:"shell"`
}
