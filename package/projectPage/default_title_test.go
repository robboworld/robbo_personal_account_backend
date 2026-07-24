package projectPage

import "testing"

func TestDefaultProjectTitle(t *testing.T) {
	cases := map[string]string{
		"ru":      "Безымянный",
		"RU":      "Безымянный",
		"zh":      "未命名",
		"zh-CN":   "未命名",
		"en":      "Untitled",
		"en-US":   "Untitled",
		"":        "Безымянный",
		"fr":      "Безымянный",
		"unknown": "Безымянный",
	}
	for input, want := range cases {
		if got := DefaultProjectTitle(input); got != want {
			t.Errorf("DefaultProjectTitle(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestLocaleFromAcceptLanguage(t *testing.T) {
	cases := map[string]string{
		"":                         DefaultLanguage,
		"ru":                       "ru",
		"ru-RU,ru;q=0.9,en;q=0.8":  "ru",
		"zh-CN,zh;q=0.9":           "zh",
		"en-US,en;q=0.9":           "en",
		"fr-FR,fr;q=0.9,en;q=0.8":  "en",
		"fr,de":                    DefaultLanguage,
		"*":                        DefaultLanguage,
	}
	for input, want := range cases {
		if got := LocaleFromAcceptLanguage(input); got != want {
			t.Errorf("LocaleFromAcceptLanguage(%q) = %q, want %q", input, got, want)
		}
	}
}
