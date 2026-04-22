package common

import "golang.org/x/crypto/bcrypt"

// BcryptCost usado em HashPassword. 10 é default padrão (suficiente pra dev e
// razoável em produção). Ajuste pra cima em prod se o hardware aguentar.
const BcryptCost = 10

// HashPassword gera hash bcrypt de senha em clear text.
// Retorna string pronta pra armazenar em users.password_hash.
func HashPassword(plain string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(plain), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// CheckPassword verifica senha clear text contra hash armazenado.
// Retorna nil se bate, erro caso contrário (inclusive ErrMismatchedHashAndPassword).
func CheckPassword(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
