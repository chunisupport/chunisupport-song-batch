package registry

import (
	"os"
	"strings"
	"testing"
)

func TestRequireNonEmptyEnv(t *testing.T) {
	// 環境変数が欠落しているケース
	t.Run("missing environment variable", func(t *testing.T) {
		const key = "TEST_REQUIRE_NON_EMPTY_ENV_MISSING"
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("failed to unset env: %v", err)
		}

		_, err := requireNonEmptyEnv(key)
		if err == nil || !strings.Contains(err.Error(), "not set") {
			t.Fatalf("expected missing environment variable error, got %v", err)
		}
	})

	// 環境変数が空のケース
	t.Run("blank environment variable", func(t *testing.T) {
		const key = "TEST_REQUIRE_NON_EMPTY_ENV_BLANK"
		t.Setenv(key, "   ")

		_, err := requireNonEmptyEnv(key)
		if err == nil || !strings.Contains(err.Error(), "must not be empty") {
			t.Fatalf("expected empty environment variable error, got %v", err)
		}
	})

	// 有効な環境変数のケース
	t.Run("valid environment variable", func(t *testing.T) {
		const key = "TEST_REQUIRE_NON_EMPTY_ENV_VALID"
		expected := "value"
		t.Setenv(key, expected)

		value, err := requireNonEmptyEnv(key)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if value != expected {
			t.Fatalf("expected %q, got %q", expected, value)
		}
	})
}
