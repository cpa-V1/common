package common

import "testing"

func TestHashPassword_RoundtripOK(t *testing.T) {
	hash, err := HashPassword("admin")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "" {
		t.Fatal("hash vazio")
	}
	if err := CheckPassword(hash, "admin"); err != nil {
		t.Errorf("senha correta rejeitada: %v", err)
	}
}

func TestCheckPassword_SenhaErradaFalha(t *testing.T) {
	hash, _ := HashPassword("senha123")
	if err := CheckPassword(hash, "wrong"); err == nil {
		t.Error("senha errada aceita — deveria falhar")
	}
}

func TestCheckPassword_HashInvalido(t *testing.T) {
	if err := CheckPassword("nao-eh-bcrypt", "qq"); err == nil {
		t.Error("hash inválido aceito — deveria falhar")
	}
}
