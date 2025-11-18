package validation

import (
	"strings"	
)

// ExtractChaveFromID: Extrai os 44 dígitos da chave de acesso do atributo Id (ex: NFe3523...)
func ExtractChaveFromID(id string) string {
	id = strings.TrimSpace(id)
	if strings.HasPrefix(id, "NFe") && len(id) == 47 {
		return id[3:] // tira "NFe" e deixa só os 44 dígitos
	}
	return ""
}

// OnlyDigits: Remove tudo que não for dígito
func OnlyDigits(s string) string {
	var out []rune
	for _, r := range s {
		if r >= '0' && r <= '9' {
			out = append(out, r)
		}
	}
	return string(out)
}

// ChooseFirstNonEmpty: Retorna o primeiro valor não vazio de uma lista
func ChooseFirstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}