package nfe

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
)

// ParsearXML faz o parse de um XML de NF-e e retorna os dados estruturados
//
// Não valida XSD nem consulta SEFAZ. Apenas extrai os dados do XML.
//
// Suporta os formatos:
//   - procNFe (XML completo com protocolo)
//   - NFe (XML da nota sem protocolo)
//
// Parâmetros:
//   - xmlData: bytes do XML
//
// Retorna:
//   - DadosNFe com os principais dados extraídos
//   - erro se o XML não puder ser parseado
//
// Exemplo:
//
//	xmlData, _ := os.ReadFile("nota.xml")
//	dados, err := nfe.ParsearXML(xmlData)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Emitente: %s\n", dados.Emitente.Nome)
//	fmt.Printf("Valor: R$ %s\n", dados.ValorTotal)
func ParsearXML(xmlData []byte) (*DadosNFe, error) {
	nfe, err := ParseNFe(xmlData)
	if err != nil {
		return nil, fmt.Errorf("falha ao parsear XML: %w", err)
	}

	return convertNFeData(nfe), nil
}

// ParsearXMLFile faz o parse de um arquivo XML
//
// Combina leitura do arquivo + parse em uma única chamada.
//
// Exemplo:
//
//	dados, err := nfe.ParsearXMLFile("nota.xml")
//	if err != nil {
//	    log.Fatal(err)
//	}
func ParsearXMLFile(xmlPath string) (*DadosNFe, error) {
	xmlData, err := os.ReadFile(xmlPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo XML: %w", err)
	}

	return ParsearXML(xmlData)
}

// ParseNFe faz o parse do XML bruto para a estrutura NFeEnvelope
//
// Tenta primeiro como procNFe (formato mais comum), depois como NFe puro.
//
// Esta é uma função de nível mais baixo. Use ParsearXML() para casos comuns.
func ParseNFe(xmlData []byte) (*NFeEnvelope, error) {
	// 1) Tentar parsear como procNFe (XML completo com protocolo)
	var proc ProcNFe
	if err := xml.Unmarshal(xmlData, &proc); err == nil && proc.NFe.InfNFe.ID != "" {
		return &proc.NFe, nil
	}

	// 2) Tentar parsear como NFe direto (sem protocolo)
	var nfe NFeEnvelope
	if err := xml.Unmarshal(xmlData, &nfe); err != nil {
		return nil, fmt.Errorf("falha ao parsear XML: não é um formato NFe válido: %w", err)
	}

	// Validar se tem o campo obrigatório
	if nfe.InfNFe.ID == "" {
		return nil, fmt.Errorf("infNFe.Id não encontrado no XML")
	}

	return &nfe, nil
}

// ExtrairChave extrai a chave de acesso de 44 dígitos do XML
//
// Aceita tanto o ID completo (ex: "NFe35250732409620000175550010000037471011544648")
// quanto apenas os 44 dígitos
//
// Exemplo:
//
//	xmlData, _ := os.ReadFile("nota.xml")
//	chave, err := nfe.ExtrairChave(xmlData)
//	fmt.Println(chave) // 35250732409620000175550010000037471011544648
func ExtrairChave(xmlData []byte) (string, error) {
	nfe, err := ParseNFe(xmlData)
	if err != nil {
		return "", err
	}

	chave := ExtractChaveFromID(nfe.InfNFe.ID)
	if chave == "" {
		return "", fmt.Errorf("não foi possível extrair a chave de acesso")
	}

	return chave, nil
}

// ExtrairChaveFromID extrai os 44 dígitos da chave do atributo Id
//
// Remove o prefixo "NFe" se presente.
//
// Exemplo:
//
//	chave := nfe.ExtractChaveFromID("NFe35250732409620000175550010000037471011544648")
//	fmt.Println(chave) // 35250732409620000175550010000037471011544648
func ExtractChaveFromID(id string) string {
	id = strings.TrimSpace(id)
	if strings.HasPrefix(id, "NFe") && len(id) == 47 {
		return id[3:] // Remove "NFe" e retorna os 44 dígitos
	}
	// Se já tem 44 dígitos, retorna como está
	if len(id) == 44 {
		return id
	}
	return ""
}

// OnlyDigits remove todos os caracteres que não são dígitos
//
// Útil para limpar chaves de acesso copiadas com formatação
//
// Exemplo:
//
//	chave := nfe.OnlyDigits("3525 0732 4096 2000 0175 5500 1000 0037 4710 1154 4648")
//	fmt.Println(chave) // 35250732409620000175550010000037471011544648
func OnlyDigits(s string) string {
	var out []rune
	for _, r := range s {
		if r >= '0' && r <= '9' {
			out = append(out, r)
		}
	}
	return string(out)
}

// ChooseFirstNonEmpty retorna o primeiro valor não vazio de uma lista
//
// Útil para escolher entre CNPJ/CPF ou outros campos opcionais
//
// Exemplo:
//
//	doc := nfe.ChooseFirstNonEmpty(dest.CNPJ, dest.CPF)
func ChooseFirstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// ValidarChaveAcesso valida o formato de uma chave de acesso
//
// Verifica:
//   - Tem exatamente 44 dígitos
//   - Contém apenas números
//   - Dígito verificador está correto
//
// Retorna erro descritivo se inválida
//
// Exemplo:
//
//	err := nfe.ValidarChaveAcesso("35250732409620000175550010000037471011544648")
//	if err != nil {
//	    log.Fatal("Chave inválida:", err)
//	}
func ValidarChaveAcesso(chave string) error {
	// Limpar espaços
	chave = strings.TrimSpace(chave)

	// Verificar tamanho
	if len(chave) != 44 {
		return fmt.Errorf("chave deve ter exatamente 44 dígitos (tem %d)", len(chave))
	}

	// Verificar se são apenas números
	for _, c := range chave {
		if c < '0' || c > '9' {
			return fmt.Errorf("chave deve conter apenas números")
		}
	}

	// Validar dígito verificador (último dígito)
	if !validarDigitoVerificador(chave) {
		return fmt.Errorf("dígito verificador inválido")
	}

	return nil
}

// validarDigitoVerificador valida o último dígito da chave (módulo 11)
func validarDigitoVerificador(chave string) bool {
	if len(chave) != 44 {
		return false
	}

	// Pegar os primeiros 43 dígitos
	base := chave[:43]
	dvEsperado := chave[43]

	// Calcular módulo 11
	multiplicador := 2
	soma := 0

	// Da direita para esquerda
	for i := len(base) - 1; i >= 0; i-- {
		digito := int(base[i] - '0')
		soma += digito * multiplicador
		multiplicador++
		if multiplicador > 9 {
			multiplicador = 2
		}
	}

	resto := soma % 11
	var dvCalculado int
	if resto == 0 || resto == 1 {
		dvCalculado = 0
	} else {
		dvCalculado = 11 - resto
	}

	return dvCalculado == int(dvEsperado-'0')
}

// convertNFeData converte a struct interna NFeEnvelope para DadosNFe público
func convertNFeData(nfe *NFeEnvelope) *DadosNFe {
	return &DadosNFe{
		Modelo: nfe.InfNFe.Ide.Modelo,
		Serie:  nfe.InfNFe.Ide.Serie,
		Numero: nfe.InfNFe.Ide.NumNf,
		Emitente: Empresa{
			Documento: nfe.InfNFe.Emit.CNPJ,
			Nome:      nfe.InfNFe.Emit.XNome,
		},
		Destinatario: Empresa{
			Documento: ChooseFirstNonEmpty(nfe.InfNFe.Dest.CNPJ, nfe.InfNFe.Dest.CPF),
			Nome:      nfe.InfNFe.Dest.XNome,
		},
		ValorTotal: nfe.InfNFe.Total.ICMSTot.VNF,
	}
}