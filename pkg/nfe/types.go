package nfe

import "encoding/xml"

// ======================================================================
// TIPOS DE RESULTADO DA VALIDAÇÃO
// ======================================================================

// ValidationResult representa o resultado completo da validação de uma NF-e
type ValidationResult struct {
	// ChaveAcesso é a chave de 44 dígitos da NF-e
	ChaveAcesso string `json:"chave_acesso,omitempty"`

	// ValidoXSD indica se o XML passou na validação XSD
	// false quando não aplicável (ex: validação apenas por chave)
	ValidoXSD bool `json:"valido_xsd"`

	// Autorizado indica se a NF-e está autorizada pela SEFAZ
	Autorizado bool `json:"autorizado"`

	// Status contém o código e mensagem retornados pela SEFAZ
	Status StatusSefaz `json:"status"`

	// DadosNFe contém os dados extraídos do XML (quando disponível)
	DadosNFe *DadosNFe `json:"dados_nfe,omitempty"`

	// Erro contém qualquer erro ocorrido durante a validação
	Erro error `json:"erro,omitempty"`
}

// StatusSefaz representa o status retornado pela SEFAZ
type StatusSefaz struct {
	// Codigo é o cStat retornado pela SEFAZ
	// Exemplos:
	//   - "100": Autorizado o uso da NF-e
	//   - "101": Cancelamento de NF-e homologado
	//   - "110": Uso Denegado
	//   - "217": NF-e não consta na base de dados da SEFAZ
	Codigo string `json:"codigo"`

	// Mensagem é o xMotivo retornado pela SEFAZ
	Mensagem string `json:"mensagem"`
}

// DadosNFe contém os principais dados extraídos de uma NF-e
type DadosNFe struct {
	// Modelo da NF-e (55 = NF-e, 65 = NFC-e)
	Modelo string `json:"modelo"`

	// Serie da nota
	Serie string `json:"serie"`

	// Numero da nota
	Numero string `json:"numero"`

	// Emitente contém os dados de quem emitiu a nota
	Emitente Empresa `json:"emitente"`

	// Destinatario contém os dados de quem recebeu a nota
	Destinatario Empresa `json:"destinatario"`

	// ValorTotal é o valor total da nota fiscal
	ValorTotal string `json:"valor_total"`
}

// Empresa representa os dados de uma empresa (emitente ou destinatário)
type Empresa struct {
	// Documento é o CNPJ ou CPF
	Documento string `json:"documento"`

	// Nome é a razão social ou nome
	Nome string `json:"nome"`
}

// ======================================================================
// STRUCTS DO XML DA NF-E (PARA PARSE)
// ======================================================================

// ProcNFe representa o XML completo procNFe (nota + protocolo)
// É o formato mais comum retornado pela SEFAZ após autorização
type ProcNFe struct {
	XMLName xml.Name    `xml:"nfeProc"`
	NFe     NFeEnvelope `xml:"NFe"`
}

// NFeEnvelope é o envelope principal da NF-e
type NFeEnvelope struct {
	XMLName xml.Name `xml:"NFe"`
	InfNFe  InfNFe   `xml:"infNFe"`
}

// InfNFe contém as informações principais da nota
type InfNFe struct {
	ID    string `xml:"Id,attr"` // Ex: "NFe35250732409620000175550010000037471011544648"
	Ide   Ide    `xml:"ide"`
	Emit  Emit   `xml:"emit"`
	Dest  Dest   `xml:"dest"`
	Total Total  `xml:"total"`
}

// Ide contém dados de identificação da nota
type Ide struct {
	Modelo string `xml:"mod"`   // 55 = NF-e, 65 = NFC-e
	Serie  string `xml:"serie"` // Série da nota
	NumNf  string `xml:"nNF"`   // Número da nota
}

// Emit representa o emitente da nota
type Emit struct {
	CNPJ  string `xml:"CNPJ"`
	XNome string `xml:"xNome"`
}

// Dest representa o destinatário da nota
type Dest struct {
	CNPJ  string `xml:"CNPJ"` // Pode estar vazio se for CPF
	CPF   string `xml:"CPF"`  // Pode estar vazio se for CNPJ
	XNome string `xml:"xNome"`
}

// Total contém os totais da nota
type Total struct {
	ICMSTot ICMSTot `xml:"ICMSTot"`
}

// ICMSTot contém o total de ICMS e valor total da NF
type ICMSTot struct {
	VNF string `xml:"vNF"` // Valor total da nota
}

// ======================================================================
// CONSTANTES DE STATUS SEFAZ
// ======================================================================

// Códigos de status mais comuns retornados pela SEFAZ
const (
	// StatusAutorizado indica que a NF-e está autorizada (cStat 100)
	StatusAutorizado = "100"

	// StatusCancelado indica que a NF-e foi cancelada (cStat 101)
	StatusCancelado = "101"

	// StatusDenegado indica uso denegado (cStat 110)
	// Emitente irregular no cadastro
	StatusDenegado = "110"

	// StatusInutilizado indica numeração inutilizada (cStat 102)
	StatusInutilizado = "102"

	// StatusNaoEncontrado indica que a NF-e não existe na base (cStat 217)
	StatusNaoEncontrado = "217"

	// StatusRejeicao indica rejeição genérica (vários códigos 2xx, 3xx, 4xx, 5xx)
	// Use o campo Mensagem para detalhes específicos
)

// ======================================================================
// MÉTODOS AUXILIARES
// ======================================================================

// IsAutorizado retorna true se o status indica autorização válida
func (s StatusSefaz) IsAutorizado() bool {
	return s.Codigo == StatusAutorizado
}

// IsCancelado retorna true se o status indica cancelamento homologado
func (s StatusSefaz) IsCancelado() bool {
	return s.Codigo == StatusCancelado
}

// IsDenegado retorna true se o status indica denegação
func (s StatusSefaz) IsDenegado() bool {
	return s.Codigo == StatusDenegado
}

// IsNaoEncontrado retorna true se a NF-e não foi encontrada na base
func (s StatusSefaz) IsNaoEncontrado() bool {
	return s.Codigo == StatusNaoEncontrado
}

// IsRejeitado retorna true se o status indica alguma rejeição
// Códigos que começam com 2, 3, 4, 5, 6 geralmente são rejeições
func (s StatusSefaz) IsRejeitado() bool {
	if len(s.Codigo) == 0 {
		return false
	}
	first := s.Codigo[0]
	return first >= '2' && first <= '6'
}

// IsValido retorna true se a nota está autorizada ou cancelada
// (ambos são status válidos - cancelada ainda consta na base)
func (s StatusSefaz) IsValido() bool {
	return s.IsAutorizado() || s.IsCancelado()
}