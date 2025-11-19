package nfe

import (
	"fmt"
	"os"

	xsdvalidate "github.com/terminalstatic/go-xsd-validate"
)

// ValidarApenasXSD valida um XML de NF-e apenas contra o schema XSD
//
// Esta é uma validação local e rápida que não consulta a SEFAZ.
// Perfeita para desenvolvimento de emissores ou validação prévia.
//
// Parâmetros:
//   - xmlData: bytes do XML a ser validado
//   - xsdPath: caminho do arquivo XSD (schema)
//
// Retorna:
//   - nil se o XML é válido
//   - erro descritivo se o XML é inválido ou se o XSD não foi encontrado
//
// Exemplo:
//
//	xmlData, _ := os.ReadFile("nota.xml")
//	err := nfe.ValidarApenasXSD(xmlData, "schemas/v4/procNFe_v4.00.xsd")
//	if err != nil {
//	    log.Fatal("XML inválido:", err)
//	}
func ValidarApenasXSD(xmlData []byte, xsdPath string) error {
	return ValidateWithXSD(xmlData, xsdPath)
}

// ValidateWithXSD é um alias para ValidarApenasXSD (mantido por compatibilidade)
func ValidateWithXSD(xmlData []byte, schemaPath string) error {
	// Verificar se o XSD existe
	if _, err := os.Stat(schemaPath); err != nil {
		return fmt.Errorf("arquivo XSD não encontrado em '%s': %w", schemaPath, err)
	}

	// Inicializa libxml2 wrapper
	xsdvalidate.Init()
	defer xsdvalidate.Cleanup()

	// Carrega o XSD
	xsdHandler, err := xsdvalidate.NewXsdHandlerUrl(schemaPath, xsdvalidate.ParsErrDefault)
	if err != nil {
		return fmt.Errorf("erro ao carregar XSD '%s': %w", schemaPath, err)
	}
	defer xsdHandler.Free()

	// Valida o XML contra o XSD
	err = xsdHandler.ValidateMem(xmlData, xsdvalidate.ValidErrDefault)
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

// ValidarXMLFile valida um arquivo XML diretamente
//
// Combina leitura do arquivo + validação XSD em uma única chamada.
//
// Exemplo:
//
//	err := nfe.ValidarXMLFile("nota.xml", "schemas/v4/procNFe_v4.00.xsd")
//	if err != nil {
//	    log.Fatal(err)
//	}
func ValidarXMLFile(xmlPath, xsdPath string) error {
	xmlData, err := os.ReadFile(xmlPath)
	if err != nil {
		return fmt.Errorf("erro ao ler arquivo XML: %w", err)
	}

	return ValidateWithXSD(xmlData, xsdPath)
}

// ValidarLote valida múltiplos XMLs contra o mesmo schema
//
// Útil para validar em batch. Retorna um map com os resultados:
// - chave: caminho do arquivo
// - valor: erro (nil se válido)
//
// Exemplo:
//
//	arquivos := []string{"nota1.xml", "nota2.xml", "nota3.xml"}
//	resultados := nfe.ValidarLote(arquivos, "schemas/v4/procNFe_v4.00.xsd")
//	
//	for arquivo, err := range resultados {
//	    if err != nil {
//	        fmt.Printf("❌ %s: %v\n", arquivo, err)
//	    } else {
//	        fmt.Printf("✅ %s: válido\n", arquivo)
//	    }
//	}
func ValidarLote(xmlPaths []string, xsdPath string) map[string]error {
	resultados := make(map[string]error)

	for _, xmlPath := range xmlPaths {
		err := ValidarXMLFile(xmlPath, xsdPath)
		resultados[xmlPath] = err
	}

	return resultados
}