package validation

import (
	"fmt"
	"os"

	xsdvalidate "github.com/terminalstatic/go-xsd-validate"
)

func ValidateWithXSD(xmlBytes []byte, schemaPath string) error {
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
