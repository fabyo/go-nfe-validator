package validation

import (
	"encoding/xml"
	"fmt"	
)

// ParseNFe: Tenta parsear como nfeProc (procNFe), depois como NFe direto
func ParseNFe(xmlBytes []byte) (*NFeEnvelope, error) {
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
