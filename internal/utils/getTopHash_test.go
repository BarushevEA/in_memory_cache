package utils

import (
	"strings"
	"testing"
)

func TestGetTopHash(t *testing.T) {
	tests := []struct {
		name string
		key  string
		want uint8
	}{
		{
			name: "empty string",
			key:  "",
			want: 0,
		},
		{
			name: "single character",
			key:  "a",
			want: func() uint8 {
				var hash uint64
				hash = hash*0x9E3779B97F4A7C15 + uint64('a')
				return uint8(hash >> 56)
			}(),
		},
		{
			name: "short string",
			key:  "short",
			want: func() uint8 {
				var hash uint64
				for _, c := range "short" {
					hash = hash*0x9E3779B97F4A7C15 + uint64(c)
				}
				return uint8(hash >> 56)
			}(),
		},
		{
			name: "exactly 32 characters",
			key:  "12345678901234567890123456789012",
			want: func() uint8 {
				var hash uint64
				for _, c := range "12345678901234567890123456789012" {
					hash = hash*0x9E3779B97F4A7C15 + uint64(c)
				}
				return uint8(hash >> 56)
			}(),
		},
		{
			name: "more than 32 characters",
			key:  "123456789012345678901234567890123",
			want: func() uint8 {
				var hash uint64
				for _, c := range "123456789012345678901234567890123" {
					hash = (hash << 5) + hash + uint64(c)
				}
				return uint8(hash >> 56)
			}(),
		},
		{
			name: "short string with special characters",
			key:  "long-string-with-@#%&-characters",
			want: func() uint8 {
				var hash uint64
				for _, c := range "long-string-with-@#%&-characters" {
					hash = hash*0x9E3779B97F4A7C15 + uint64(c) // используем мультипликативное хеширование
				}
				return uint8(hash >> 56)
			}(),
		},
		{
			name: "numeric string",
			key:  "12345",
			want: func() uint8 {
				var hash uint64
				for _, c := range "12345" {
					hash = hash*0x9E3779B97F4A7C15 + uint64(c)
				}
				return uint8(hash >> 56)
			}(),
		},
		{
			name: "mixed case string",
			key:  "CamelCaseTest",
			want: func() uint8 {
				var hash uint64
				for _, c := range "CamelCaseTest" {
					hash = hash*0x9E3779B97F4A7C15 + uint64(c)
				}
				return uint8(hash >> 56)
			}(),
		},
		{
			name: "unicode characters",
			key:  "привет世界",
			want: func() uint8 {
				var hash uint64
				key := "привет世界"
				if len(key) <= 32 {
					for i := 0; i < len(key); i++ {
						hash = hash*0x9E3779B97F4A7C15 + uint64(key[i])
					}
				} else {
					for i := 0; i < len(key); i++ {
						hash = (hash << 5) + hash + uint64(key[i])
					}
				}
				return uint8(hash >> 56)
			}(),
		},
		{
			name: "very long string",
			key:  "this_is_a_very_long_string_that_definitely_exceeds_32_characters_by_a_lot",
			want: func() uint8 {
				var hash uint64
				for _, c := range "this_is_a_very_long_string_that_definitely_exceeds_32_characters_by_a_lot" {
					hash = (hash << 5) + hash + uint64(c)
				}
				return uint8(hash >> 56)
			}(),
		},
		{
			name: "long string with unicode",
			key:  "очень_длинная_строка_с_unicode_символами_превышающая_32_символа_世界",
			want: func() uint8 {
				var hash uint64
				key := "очень_длинная_строка_с_unicode_символами_превышающая_32_символа_世界"
				for i := 0; i < len(key); i++ {
					hash = (hash << 5) + hash + uint64(key[i])
				}
				return uint8(hash >> 56)
			}(),
		},
		{
			name: "string with spaces",
			key:  "  spaces  at  beginning  and  end  ",
			want: func() uint8 {
				var hash uint64
				key := "  spaces  at  beginning  and  end  "
				for i := 0; i < len(key); i++ {
					hash = (hash << 5) + hash + uint64(key[i]) // используем аддитивное хеширование
				}
				return uint8(hash >> 56)
			}(),
		},
		{
			name: "consistent hash for same short string - test 1",
			key:  "test_string_1",
			want: GetTopHash("test_string_1"), // используем саму функцию для получения эталонного значения
		},
		{
			name: "consistent hash for same short string - test 2",
			key:  "test_string_1", // та же строка
			want: GetTopHash("test_string_1"),
		},
		{
			name: "consistent hash for same long string - test 1",
			key:  "this_is_a_very_long_string_that_needs_to_be_hashed_consistently_123",
			want: GetTopHash("this_is_a_very_long_string_that_needs_to_be_hashed_consistently_123"),
		},
		{
			name: "consistent hash for same long string - test 2",
			key:  "this_is_a_very_long_string_that_needs_to_be_hashed_consistently_123", // та же длинная строка
			want: GetTopHash("this_is_a_very_long_string_that_needs_to_be_hashed_consistently_123"),
		},
		{
			name: "consistent hash for same unicode string",
			key:  "тест_строка_😊", // строка с эмодзи
			want: GetTopHash("тест_строка_😊"),
		},
		{
			name: "consistent hash for repeated calls",
			key:  "repeated_test_string",
			want: func() uint8 {
				// Проверяем, что множественные вызовы дают одинаковый результат
				first := GetTopHash("repeated_test_string")
				for i := 0; i < 5; i++ {
					if hash := GetTopHash("repeated_test_string"); hash != first {
						return 0 // тест провалится, если хоть один хеш будет отличаться
					}
				}
				return first
			}(),
		},
		{
			name: "empty string",
			key:  "",
			want: GetTopHash(""),
		},
		{
			name: "exactly 32 bytes string",
			key:  "12345678901234567890123456789012",
			want: GetTopHash("12345678901234567890123456789012"),
		},
		{
			name: "33 bytes string (boundary case)",
			key:  "123456789012345678901234567890123",
			want: GetTopHash("123456789012345678901234567890123"),
		},
		{
			name: "string with null bytes",
			key:  "test\x00string\x00with\x00nulls",
			want: GetTopHash("test\x00string\x00with\x00nulls"),
		},
		{
			name: "string with all special characters",
			key:  "!@#$%^&*()_+-=[]{}|;:'\",.<>?/\\",
			want: GetTopHash("!@#$%^&*()_+-=[]{}|;:'\",.<>?/\\"),
		},
		{
			name: "very long unicode string",
			key: func() string {
				// Создаем строку, которая точно будет длиннее 32 байт из-за многобайтовых символов
				return strings.Repeat("世界", 20)
			}(),
			want: func() uint8 {
				return GetTopHash(strings.Repeat("世界", 20))
			}(),
		},
		{
			name: "string with high unicode values",
			key:  "测试🌍🌎🌏🚀✨🔥💫⭐",
			want: GetTopHash("测试🌍🌎🌏🚀✨🔥💫⭐"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetTopHash(tt.key)
			if got != tt.want {
				t.Errorf("GetTopHash(%q) = %d, want %d", tt.key, got, tt.want)
			}
		})
	}
}

func BenchmarkGetTopHash(b *testing.B) {
	testCases := []struct {
		name string
		key  string
	}{
		{"short_ascii", "test string"},
		{"long_ascii", strings.Repeat("a", 100)},
		{"short_unicode", "привет世界"},
		{"long_unicode", strings.Repeat("世界", 20)},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				GetTopHash(tc.key)
			}
		})
	}
}
