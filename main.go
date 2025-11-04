package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"os"
	"strings"

	xsdvalidate "github.com/terminalstatic/go-xsd-validate"
)

//
// ======== Structs de NFe (simplificados) ========
//

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

//
// ======== Structs da resposta JSON ========
//

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

//
// ======== main ========
//

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Uso:")
		fmt.Println("  go run main.go <caminho_nfe.xml> <caminho_schema.xsd>")
		os.Exit(1)
	}

	xmlPath := os.Args[1]
	xsdPath := os.Args[2]

	xmlBytes, err := os.ReadFile(xmlPath)
	if err != nil {
		printJSONAndExit(ValidationResponse{
			Tipo: "nfe",
			Erro: fmt.Sprintf("Erro ao ler XML: %v", err),
		})
	}

	// 1) Validação XSD usando go-xsd-validate
	if err := validateWithXSD(xmlBytes, xsdPath); err != nil {
		printJSONAndExit(ValidationResponse{
			Tipo:      "nfe",
			ValidoXSD: false,
			Erro:      fmt.Sprintf("XML inválido contra XSD: %v", err),
		})
	}

	// 2) Parse do XML (tenta primeiro nfeProc, depois NFe direto)
	nfeEnv, err := parseNFe(xmlBytes)
	if err != nil {
		printJSONAndExit(ValidationResponse{
			Tipo:      "nfe",
			ValidoXSD: true,
			Erro:      fmt.Sprintf("Erro ao parsear NFe: %v", err),
		})
	}

	chave := extractChaveFromID(nfeEnv.InfNFe.ID)

	dados := &DadosXMLNFe{
		Modelo:       nfeEnv.InfNFe.Ide.Modelo,
		Serie:        nfeEnv.InfNFe.Ide.Serie,
		Numero:       nfeEnv.InfNFe.Ide.NumNf,
		EmitCNPJ:     onlyDigits(nfeEnv.InfNFe.Emit.CNPJ),
		EmitRazao:    nfeEnv.InfNFe.Emit.XNome,
		DestDoc:      chooseFirstNonEmpty(onlyDigits(nfeEnv.InfNFe.Dest.CNPJ), onlyDigits(nfeEnv.InfNFe.Dest.CPF)),
		DestNome:     nfeEnv.InfNFe.Dest.XNome,
		ValorTotalNF: nfeEnv.InfNFe.Total.ICMSTot.VNF,
	}

	// 3) Consulta SEFAZ (mock)
	sefazStatus := consultarSefazMock(chave)

	// 4) Monta resposta JSON
	resp := ValidationResponse{
		Tipo:        "nfe",
		ChaveAcesso: chave,
		ValidoXSD:   true,
		Sefaz:       sefazStatus,
		DadosXML:    dados,
	}

	printJSONAndExit(resp)
}

//
// ======== XSD usando go-xsd-validate ========
//

func validateWithXSD(xmlBytes []byte, schemaPath string) error {
	// opcional: checar se o XSD existe, pra erro ficar mais claro
	if _, err := os.Stat(schemaPath); err != nil {
		return fmt.Errorf("arquivo XSD não encontrado em '%s': %w", schemaPath, err)
	}

	// Inicializa libxml2 wrapper
	xsdvalidate.Init()
	defer xsdvalidate.Cleanup()

	// Carrega o XSD (como no exemplo da doc)
	xsdHandler, err := xsdvalidate.NewXsdHandlerUrl(schemaPath, xsdvalidate.ParsErrDefault)
	if err != nil {
		return fmt.Errorf("erro ao carregar XSD '%s': %w", schemaPath, err)
	}
	defer xsdHandler.Free()

	// Option 2 do exemplo: validar direto da memória
	err = xsdHandler.ValidateMem(xmlBytes, xsdvalidate.ValidErrDefault)
	if err != nil {
		switch e := err.(type) {
		case xsdvalidate.ValidationError:
			if len(e.Errors) > 0 {
				first := e.Errors[0]
				return fmt.Errorf("falha na validação XSD (linha %d): %s", first.Line, first.Message)
			}
			return fmt.Errorf("falha na validação XSD: %v", e)
		default:
			return fmt.Errorf("erro de validação XSD: %w", err)
		}
	}

	return nil
}

//
// ======== parse NFe / nfeProc ========
//

func parseNFe(xmlBytes []byte) (*NFeEnvelope, error) {
	// 1) tenta nfeProc (procNFe)
	var proc ProcNFe
	if err := xml.Unmarshal(xmlBytes, &proc); err == nil && proc.NFe.InfNFe.ID != "" {
		return &proc.NFe, nil
	}

	// 2) tenta NFe direto
	var nfe NFeEnvelope
	if err := xml.Unmarshal(xmlBytes, &nfe); err != nil {
		return nil, err
	}
	if nfe.InfNFe.ID == "" {
		return nil, fmt.Errorf("infNFe.Id não encontrado")
	}
	return &nfe, nil
}

//
// ======== helpers ========
//

func extractChaveFromID(id string) string {
	id = strings.TrimSpace(id)
	if strings.HasPrefix(id, "NFe") && len(id) == 47 {
		return id[3:] // tira "NFe" e deixa só os 44 dígitos
	}
	return ""
}

// plugar a consulta real (gateway, SEFAZ, etc.)
func consultarSefazMock(chave string) SefazStatus {
	if chave == "" {
		return SefazStatus{
			Autorizado: false,
			Codigo:     "000",
			Mensagem:   "Chave de acesso não encontrada no XML",
		}
	}

	// MOCK 
	return SefazStatus{
		Autorizado: true,
		Codigo:     "100",
		Mensagem:   "Autorizado o uso da NF-e (mock)",
	}
}

func onlyDigits(s string) string {
	var out []rune
	for _, r := range s {
		if r >= '0' && r <= '9' {
			out = append(out, r)
		}
	}
	return string(out)
}

func chooseFirstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

func printJSONAndExit(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Println(`{"erro": "falha ao gerar JSON"}`)
		os.Exit(1)
	}
	fmt.Println(string(b))
	os.Exit(0)
}
