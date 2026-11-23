package contract

import (
	"embed"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/krewire/krewire/pkg/model"
)

// SchemaDialect is the JSON Schema dialect of the published schemas.
const SchemaDialect = "https://json-schema.org/draft/2020-12/schema"

// SchemaBaseURL prefixes the $id of every published schema.
const SchemaBaseURL = "https://krewire.dev/schemas/toolchain/" + Version + "/"

//go:embed schemas/*.json
var publishedSchemas embed.FS

// kindTypes binds each document kind to its Go type.
var kindTypes = map[Kind]reflect.Type{
	KindProjectContext:  reflect.TypeOf(ProjectContext{}),
	KindTaskBrief:       reflect.TypeOf(TaskBrief{}),
	KindPlan:            reflect.TypeOf(Plan{}),
	KindPolicy:          reflect.TypeOf(Policy{}),
	KindApprovalRecord:  reflect.TypeOf(ApprovalRecord{}),
	KindExecutionRecord: reflect.TypeOf(ExecutionRecord{}),
	KindEvidenceReport:  reflect.TypeOf(EvidenceReport{}),
	KindCommandCatalog:  reflect.TypeOf(CommandCatalog{}),
}

func strs[T ~string](vs ...T) []string {
	out := make([]string, len(vs))
	for i, v := range vs {
		out[i] = string(v)
	}
	return out
}

// enums lists closed value sets. Values come from the constants themselves,
// so a schema cannot list a value the Go code does not define.
var enums = map[reflect.Type][]string{
	reflect.TypeOf(Kind("")):        strs(Kinds...),
	reflect.TypeOf(Risk("")):        strs(RiskLow, RiskMedium, RiskHigh),
	reflect.TypeOf(Severity("")):    strs(SeverityError, SeverityWarning, SeverityInfo),
	reflect.TypeOf(ToolName("")):    strs(ToolWriteFile, ToolDeleteFile, ToolRunCommand, ToolFetch),
	reflect.TypeOf(ExecStatus("")):  strs(ExecApplied, ExecDenied, ExecFailed),
	reflect.TypeOf(Stage("")):       strs(Stages...),
	reflect.TypeOf(StageStatus("")): strs(StageOK, StageFailed, StageSkipped),
	reflect.TypeOf(Outcome("")): strs(OutcomeVerified, OutcomeVerificationFailed, OutcomeDenied,
		OutcomeCancelled, OutcomeEngineUnavailable, OutcomeFailed),
	reflect.TypeOf(CommandClass("")): strs(ClassDeterministic, ClassAgentic, ClassOrchestrated),
	reflect.TypeOf(model.DiagnosticSeverity("")): strs(model.DiagMissing, model.DiagConflict,
		model.DiagStale, model.DiagMalformed),
	reflect.TypeOf(model.AutonomyLevel("")): strs(model.AutonomyAutonomous, model.AutonomyApprovalRequired,
		model.AutonomyEscalation),
	reflect.TypeOf(model.ActionCategory("")): strs(model.ActionRead, model.ActionWrite, model.ActionDelete,
		model.ActionCommand, model.ActionNetwork, model.ActionGitPush, model.ActionDeploy),
	reflect.TypeOf(model.VerificationStatus("")): strs(model.GatePlanned, model.GateAttempted, model.GatePassed,
		model.GateFailed, model.GateSkipped, model.GateBlocked),
	reflect.TypeOf(model.QualityGateType("")): strs(model.GateTypeFormat, model.GateTypeStatic, model.GateTypeTest,
		model.GateTypeBuild, model.GateTypeLayering, model.GateTypeSecurity, model.GateTypeCustom),
}

var timeType = reflect.TypeOf(time.Time{})

// GenerateSchema derives the JSON Schema of a kind from its Go type.
func GenerateSchema(k Kind) ([]byte, error) {
	t, ok := kindTypes[k]
	if !ok {
		return nil, Errorf(CodeInvalidParams, "unknown kind %q", k)
	}
	defs := map[string]any{}
	g := &schemaGen{defs: defs}
	g.ref(t)
	root := defs[t.Name()].(map[string]any)
	props := root["properties"].(map[string]any)
	props["api_version"] = map[string]any{"const": APIVersion}
	props["kind"] = map[string]any{"const": string(k)}
	doc := map[string]any{
		"$schema": SchemaDialect,
		"$id":     SchemaBaseURL + string(k) + ".json",
		"title":   string(k),
		"description": fmt.Sprintf("Krewire toolchain contract %s, %s (contract %s).",
			k, APIVersion, ContractVersion),
		"$ref":  "#/$defs/" + t.Name(),
		"$defs": defs,
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

// Schema returns the published schema of a kind, as embedded at build time.
func Schema(k Kind) ([]byte, error) {
	data, err := publishedSchemas.ReadFile("schemas/" + string(k) + ".json")
	if err != nil {
		return nil, Errorf(CodeInvalidParams, "no published schema for kind %q", k)
	}
	return data, nil
}

type schemaGen struct{ defs map[string]any }

// ref returns a schema for t, registering struct definitions in $defs.
func (g *schemaGen) ref(t reflect.Type) map[string]any {
	if t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	if vals, ok := enums[t]; ok {
		return map[string]any{"type": "string", "enum": vals}
	}
	switch {
	case t == timeType:
		return map[string]any{"type": "string", "format": "date-time"}
	case t.Kind() == reflect.Struct:
		if _, done := g.defs[t.Name()]; !done {
			g.defs[t.Name()] = map[string]any{} // break recursion
			props, required := map[string]any{}, []string{}
			g.fields(t, props, &required)
			def := map[string]any{"type": "object", "properties": props, "additionalProperties": false}
			if len(required) > 0 {
				def["required"] = required
			}
			g.defs[t.Name()] = def
		}
		return map[string]any{"$ref": "#/$defs/" + t.Name()}
	case t.Kind() == reflect.Slice:
		return map[string]any{"type": "array", "items": g.ref(t.Elem())}
	case t.Kind() == reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": g.ref(t.Elem())}
	case t.Kind() == reflect.String:
		return map[string]any{"type": "string"}
	case t.Kind() == reflect.Bool:
		return map[string]any{"type": "boolean"}
	case t.Kind() >= reflect.Int && t.Kind() <= reflect.Uint64:
		return map[string]any{"type": "integer"}
	case t.Kind() == reflect.Float32 || t.Kind() == reflect.Float64:
		return map[string]any{"type": "number"}
	}
	panic(fmt.Sprintf("contract: no schema mapping for %s", t))
}

// fields collects a struct's JSON properties, flattening untagged embeds.
func (g *schemaGen) fields(t reflect.Type, props map[string]any, required *[]string) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		tag := f.Tag.Get("json")
		if tag == "-" {
			continue
		}
		name, opts, _ := strings.Cut(tag, ",")
		if f.Anonymous && name == "" {
			g.fields(f.Type, props, required)
			continue
		}
		if name == "" {
			name = f.Name
		}
		props[name] = g.ref(f.Type)
		if !strings.Contains(opts, "omitempty") {
			*required = append(*required, name)
		}
	}
}
