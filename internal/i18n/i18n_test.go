package i18n

import (
	"reflect"
	"testing"
)

func TestParseDefaultEnglish(t *testing.T) {
	if Parse("") != En {
		t.Fatal("empty must be English")
	}
	if Parse("en") != En {
		t.Fatal("en")
	}
	if Parse("nope") != En {
		t.Fatal("unknown must be English")
	}
	if Parse("zh") != Zh {
		t.Fatal("zh")
	}
	if Parse("zh-CN") != Zh {
		t.Fatal("zh-CN")
	}
}

func TestDefaultCatalogEnglish(t *testing.T) {
	Set("")
	if C().HelpShort != en.HelpShort {
		t.Fatalf("default catalog should be English, got %q", C().HelpShort)
	}
}

func TestSetChinese(t *testing.T) {
	t.Cleanup(func() { Set("") })
	Set("zh")
	if C().HelpShort != zh.HelpShort {
		t.Fatalf("zh catalog, got %q", C().HelpShort)
	}
	Set("zh-CN")
	if Current() != Zh {
		t.Fatal("zh-CN")
	}
}

func TestFirstQuoted(t *testing.T) {
	got := firstQuoted("(\n    \"zh-Hans-CN\",\n    \"en-CN\"\n)\n")
	if got != "zh-Hans-CN" {
		t.Fatalf("quoted: %q", got)
	}
	if firstQuoted("none") != "" {
		t.Fatal("missing quote")
	}
}

func TestUsePrefersSavedLanguage(t *testing.T) {
	t.Cleanup(func() { Set("") })
	Use("en")
	if Current() != En {
		t.Fatal("saved en")
	}
	Use("zh-Hans-CN")
	if Current() != Zh {
		t.Fatal("saved zh")
	}
}

func TestCatalogComplete(t *testing.T) {
	for name, cat := range map[string]Catalog{"en": en, "zh": zh} {
		v := reflect.ValueOf(cat)
		typ := v.Type()
		for i := 0; i < v.NumField(); i++ {
			f := v.Field(i)
			if f.Kind() != reflect.String {
				continue
			}
			if f.String() == "" {
				t.Errorf("%s.%s is empty", name, typ.Field(i).Name)
			}
		}
	}
}
