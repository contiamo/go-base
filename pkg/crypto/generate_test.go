package crypto

import "testing"

func TestRandomString(t *testing.T) {
	t.Run("does not repeat the same value twice", func(t *testing.T) {
		r1, err := GenerateRandomString(16)
		if err != nil {
			t.Fatal(err)
		}
		if len(r1) != 16 {
			t.Fatal("string length must be 15 characters")
		}

		r2, err := GenerateRandomString(16)
		if err != nil {
			t.Fatal(err)
		}
		if len(r2) != 16 {
			t.Fatal("string length must be 15 characters")
		}

		if r1 == r2 {
			t.Fatal("must produce different strings in a row")
		}
	})

	t.Run("supports lengths %2 != 0", func(t *testing.T) {
		r1, err := GenerateRandomString(15)
		if err != nil {
			t.Fatal(err)
		}
		if len(r1) != 15 {
			t.Fatal("string length must be 15 characters")
		}
	})
}
