package core

import "strings"

func NormalizeExecutionEnv(v string) ExecutionEnv {
	s := strings.ToUpper(strings.TrimSpace(v))
	switch s {
	case string(ExecutionEnvProd), "PRODUCTION":
		return ExecutionEnvProd
	case string(ExecutionEnvStaging), "STAGE":
		return ExecutionEnvStaging
	case string(ExecutionEnvCanary):
		return ExecutionEnvCanary
	case string(ExecutionEnvDev), "DEVELOPMENT":
		return ExecutionEnvDev
	case "":
		return ExecutionEnvUnknown
	default:
		if s == string(ExecutionEnvUnknown) {
			return ExecutionEnvUnknown
		}
		return ExecutionEnvUnknown
	}
}

func NormalizeExecutionType(v string) ExecutionType {
	s := strings.ToUpper(strings.TrimSpace(v))
	s = strings.ReplaceAll(s, "_", "-")
	switch s {
	case string(ExecutionTypeReal), "PRODUCTION-RUN":
		return ExecutionTypeReal
	case string(ExecutionTypeTestRun), "TEST":
		return ExecutionTypeTestRun
	case string(ExecutionTypeNightly):
		return ExecutionTypeNightly
	case string(ExecutionTypeSmoke):
		return ExecutionTypeSmoke
	case "":
		return ExecutionTypeUnknown
	default:
		if s == string(ExecutionTypeUnknown) {
			return ExecutionTypeUnknown
		}
		return ExecutionTypeUnknown
	}
}

func (f ExecutionFlags) Valid() bool {
	switch f.Env {
	case ExecutionEnvUnknown, ExecutionEnvProd, ExecutionEnvStaging, ExecutionEnvCanary, ExecutionEnvDev:
	default:
		return false
	}
	switch f.Type {
	case ExecutionTypeUnknown, ExecutionTypeReal, ExecutionTypeTestRun, ExecutionTypeNightly, ExecutionTypeSmoke:
	default:
		return false
	}
	return true
}

func ExecutionFlagsFromPayload(raw map[string]interface{}) ExecutionFlags {
	if raw == nil {
		return ExecutionFlags{Env: ExecutionEnvUnknown, Type: ExecutionTypeReal}
	}
	env := stringField(raw, "execution_env", "")
	if env == "" {
		env = stringField(raw, "environment", "")
	}
	typ := stringField(raw, "execution_type", "")
	if typ == "" {
		typ = stringField(raw, "run_type", "")
	}
	flags := ExecutionFlags{
		Env:  NormalizeExecutionEnv(env),
		Type: NormalizeExecutionType(typ),
	}
	if flags.Env == ExecutionEnvUnknown && flags.Type == ExecutionTypeUnknown {
		flags.Type = ExecutionTypeReal
	} else if flags.Type == ExecutionTypeUnknown {
		flags.Type = ExecutionTypeReal
	}
	return flags
}

func mergeExecutionFlags(base, override ExecutionFlags) ExecutionFlags {
	out := base
	if override.Env != ExecutionEnvUnknown {
		out.Env = override.Env
	}
	if override.Type != ExecutionTypeUnknown {
		out.Type = override.Type
	}
	if out.Env == ExecutionEnvUnknown {
		out.Env = ExecutionEnvUnknown
	}
	if out.Type == ExecutionTypeUnknown {
		out.Type = ExecutionTypeReal
	}
	return out
}
