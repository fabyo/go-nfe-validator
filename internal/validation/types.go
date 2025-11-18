package validation

import (
	"encoding/xml"
)

// ======================================================================
// Structs de NFe (Simplificados)
// ======================================================================

// Caso o XML seja um procNFe (mais comum: arquivo final retornado pela SEFAZ)
type ProcNFe struct {
	XMLName xml.Name   `xml:"nfeProc"`
	NFe     NFeEnvelope `xml:"NFe"`
}

// NFe "pura" (root <NFe>...</NFe>)
type NFeEnvelope struct {
	XMLName xml.Name `xml:"NFe"`
	InfNFe  InfNFe   `xml:"infNFe"`
}

type InfNFe struct {
	ID   string `xml:"Id,attr"` // Id="NFe<chave>"
	Ide  Ide    `xml:"ide"`
	Emit Emit   `xml:"emit"`
	Dest Dest   `xml:"dest"`
	Total Total `xml:"total"`
}

type Ide struct {
	Modelo string `xml:"mod"`
	Serie  string `xml:"serie"`
	NumNf  string `xml:"nNF"`
}

type Emit struct {
	CNPJ  string `xml:"CNPJ"`
	XNome string `xml:"xNome"`
}

type Dest struct {
	CNPJ  string `xml:"CNPJ"`
	CPF   string `xml:"CPF"`
	XNome string `xml:"xNome"`
}

type Total struct {
	ICMSTot ICMSTot `xml:"ICMSTot"`
}

type ICMSTot struct {
	VNF string `xml:"vNF"`
}

// ======================================================================
// Structs da Resposta JSON (Modelo de Dados)
// ======================================================================

type SefazStatus struct {
	Autorizado bool   `json:"autorizado"`
	Codigo     string `json:"codigo"`
	Mensagem   string `json:"mensagem"`
}

type DadosXMLNFe struct {
	Modelo       string `json:"modelo"`
	Serie        string `json:"serie"`
	Numero       string `json:"numero"`
	EmitCNPJ     string `json:"emitente_cnpj"`
	EmitRazao    string `json:"emitente_razao"`
	DestDoc      string `json:"destinatario_doc"`
	DestNome     string `json:"destinatario_nome"`
	ValorTotalNF string `json:"valor_total_nota"`
}

type ValidationResponse struct {
	Tipo        string        `json:"tipo"` // nfe, nfce, etc.
	ChaveAcesso string        `json:"chave_acesso"`
	ValidoXSD   bool          `json:"valido_xsd"`
	Sefaz       SefazStatus   `json:"sefaz"`
	DadosXML    *DadosXMLNFe  `json:"dados_xml,omitempty"`
	Erro        string        `json:"erro,omitempty"`
}
