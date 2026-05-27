package models

func StrPtrVal(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func IntPtrVal(p *int) *int {
	return p
}

func StrPtr(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
