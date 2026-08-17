package target

import (
	"reflect"
	"testing"
)

func TestLanguageOf(t *testing.T) {
	cases := []struct {
		kind  Kind
		want  Language
		found bool
	}{
		{KindGoBinary, LanguageGo, true},
		{KindGoLibrary, LanguageGo, true},
		{KindGoTest, LanguageGo, true},
		{"go_proto_library", LanguageGo, true},
		{KindCcBinary, LanguageC, true},
		{KindCcLibrary, LanguageC, true},
		{KindCcTest, LanguageC, true},
		{"genrule", "", false},
		{"filegroup", "", false},
		{"sh_binary", "", false},
		{"pkg_rpm", "", false},
		{"", "", false},
	}
	for _, tc := range cases {
		got, found := LanguageOf(tc.kind)
		if got != tc.want || found != tc.found {
			t.Fatalf("LanguageOf(%q) = %q, %t; want %q, %t", tc.kind, got, found, tc.want, tc.found)
		}
	}
}

// A package holding both languages reports them alphabetically, so the order
// never depends on how its targets happened to be indexed.
func TestLanguagesAreAlphabetical(t *testing.T) {
	cases := []struct {
		name  string
		kinds []Kind
		want  []Language
	}{
		{"go only", []Kind{KindGoLibrary, KindGoTest}, []Language{LanguageGo}},
		{"c only", []Kind{KindCcLibrary, KindCcTest}, []Language{LanguageC}},
		{"go then c", []Kind{KindGoLibrary, KindCcLibrary}, []Language{LanguageC, LanguageGo}},
		{"c then go", []Kind{KindCcLibrary, KindGoLibrary}, []Language{LanguageC, LanguageGo}},
		{"neither", []Kind{"genrule", "filegroup"}, []Language{}},
		{"none at all", nil, []Language{}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Languages(tc.kinds); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("Languages(%v) = %v, want %v", tc.kinds, got, tc.want)
			}
		})
	}
}

func TestLanguagesDeduplicates(t *testing.T) {
	got := Languages([]Kind{KindGoLibrary, KindGoTest, KindGoBinary, KindCcTest, KindCcLibrary})
	if len(got) != 2 {
		t.Fatalf("Languages() = %v, want two distinct languages", got)
	}
}

func TestIsPackagingByRuleKind(t *testing.T) {
	for _, kind := range []Kind{"pkg_rpm", "pkg_tar", "pkg_tar_impl", "pkg_files", "pkg_deb"} {
		if !IsPackaging(Target{Kind: kind, Name: "whatever"}) {
			t.Fatalf("%q should be recognised as packaging", kind)
		}
	}
}

// The other convention in this repo: a genrule named for the artifact it wraps.
// The rule kind alone cannot distinguish those from any other genrule.
func TestIsPackagingByNamingConvention(t *testing.T) {
	for _, name := range []string{"basalt_rpm", "probe_rpm", "packer_rpm", "sonic_tgz", "thing_appimage"} {
		if !IsPackaging(Target{Kind: "genrule", Name: name}) {
			t.Fatalf("%q should be recognised as packaging", name)
		}
	}
}

func TestIsNotPackaging(t *testing.T) {
	cases := []Target{
		{Kind: KindGoBinary, Name: "basalt"},
		{Kind: KindGoLibrary, Name: "basalt_lib"},
		{Kind: KindGoTest, Name: "basalt_test"},
		{Kind: "genrule", Name: "generate_sources"},
		{Kind: KindCcBinary, Name: "sonicd"},
		{Kind: "filegroup", Name: "routing_scripts"},
	}
	for _, candidate := range cases {
		if IsPackaging(candidate) {
			t.Fatalf("%+v should not be recognised as packaging", candidate)
		}
	}
}

func BenchmarkLanguageOf(b *testing.B) {
	for b.Loop() {
		LanguageOf(KindGoLibrary)
	}
}

func BenchmarkLanguages(b *testing.B) {
	kinds := []Kind{KindGoLibrary, KindGoTest, KindCcLibrary, "genrule"}
	for b.Loop() {
		Languages(kinds)
	}
}

func BenchmarkIsPackaging(b *testing.B) {
	t := Target{Kind: "genrule", Name: "basalt_rpm"}
	for b.Loop() {
		IsPackaging(t)
	}
}
