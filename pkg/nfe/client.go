package nfe

import (
	"fmt"
	"os"

	"github.com/fabyo/go-nfe-validator/internal/config"
	"github.com/fabyo/go-nfe-validator/internal/sefaz"
	"github.com/fabyo/go-nfe-validator/internal/validation"
)

// Client é o cliente principal para validação de NF-e
type Client struct {
	sefaz *sefaz.Client
	cfg   *config.Config
}

// Config representa as configurações do cliente
type Config struct {
	// Diretório onde estão os certificados
	CertDir string
	// Nome do arquivo da chave privada (ex: "key.pem")
	CertKeyFile string
	// Nome do arquivo do certificado público (ex: "cert.pem")
	CertPubFile string
	// CNPJ da empresa (opcional)
	CNPJ string
	// Código UF IBGE (ex: "35" para SP)
	UF string
	// URL de consulta da SEFAZ (opcional, usa padrão se vazio)
	ConsultaURL string
	// URL de distribuição (opcional)
	DistURL string
	// Ambiente: "production" ou "homologation"
	Env string
}

// NewClient cria um novo cliente de validação NF-e
//
// Exemplo:
//
//	client, err := nfe.NewClient(nfe.Config{
//	    CertDir:     "cert",
//	    CertKeyFile: "key.pem",
//	    CertPubFile: "cert.pem",
//	    UF:          "35",
//	    Env:         "production",
//	})
func NewClient(cfg Config) (*Client, error) {
	// Configuração interna
	internalCfg := &config.Config{
		CertDir:     cfg.CertDir,
		CertKeyFile: cfg.CertKeyFile,
		CertPubFile: cfg.CertPubFile,
		CNPJ:        cfg.CNPJ,
		UF:          cfg.UF,
		ConsultaURL: cfg.ConsultaURL,
		DistURL:     cfg.DistURL,
		Env:         cfg.Env,
	}

	// Se não especificou ambiente, usa production
	if internalCfg.Env == "" {
		internalCfg.Env = "production"
	}

	// Criar cliente SEFAZ
	sefazClient, err := sefaz.NewClient(internalCfg)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar cliente SEFAZ: %w", err)
	}

	return &Client{
		sefaz: sefazClient,
		cfg:   internalCfg,
	}, nil
}

// NewClientFromEnv cria um cliente usando variáveis de ambiente
// Lê de .env.production ou .env.homologation automaticamente
//
// Variáveis necessárias:
//   - NFE_CERT_DIR
//   - NFE_CERT_KEY_FILE
//   - NFE_CERT_PUB_FILE
//   - NFE_UF_IBGE
//   - SEFAZ_CONSULTA_URL
//
// Exemplo:
//
//	client, err := nfe.NewClientFromEnv()
func NewClientFromEnv() (*Client, error) {
	cfg := config.Load()

	sefazClient, err := sefaz.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("falha ao criar cliente SEFAZ: %w", err)
	}

	return &Client{
		sefaz: sefazClient,
		cfg:   cfg,
	}, nil
}

// ValidarXML valida um XML de NF-e completamente (XSD + Parse + SEFAZ)
//
// Parâmetros:
//   - xmlPath: caminho do arquivo XML
//   - xsdPath: caminho do arquivo XSD (schema)
//
// Retorna ValidationResult com todos os dados e status da SEFAZ
//
// Exemplo:
//
//	result, err := client.ValidarXML("nota.xml", "schemas/v4/procNFe_v4.00.xsd")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Autorizada: %v\n", result.Autorizado)
func (c *Client) ValidarXML(xmlPath, xsdPath string) (*ValidationResult, error) {
	// 1. Validar XSD
	xmlData, err := os.ReadFile(xmlPath)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo XML: %w", err)
	}

	if err := ValidateWithXSD(xmlData, xsdPath); err != nil {
		return &ValidationResult{
			ValidoXSD: false,
			Erro:      fmt.Errorf("falha na validação XSD: %w", err),
		}, nil
	}

	// 2. Parse do XML
	nfe, err := validation.ParseNFe(xmlData)
	if err != nil {
		return &ValidationResult{
			ValidoXSD: true,
			Erro:      fmt.Errorf("falha ao parsear XML: %w", err),
		}, nil
	}

	// Extrair chave
	chave := validation.ExtractChaveFromID(nfe.InfNFe.ID)
	if chave == "" {
		chave = nfe.InfNFe.ID
	}

	// 3. Consultar SEFAZ
	status, err := c.sefaz.ConsultaSituacaoNFe(chave)
	if err != nil {
		return &ValidationResult{
			ValidoXSD:   true,
			ChaveAcesso: chave,
			DadosNFe:    convertInternalNFeData(nfe),
			Erro:        fmt.Errorf("falha na consulta SEFAZ: %w", err),
		}, nil
	}

	return &ValidationResult{
		ValidoXSD:   true,
		ChaveAcesso: chave,
		Autorizado:  status.Autorizado,
		Status: StatusSefaz{
			Codigo:   status.Codigo,
			Mensagem: status.Mensagem,
		},
		DadosNFe: convertInternalNFeData(nfe),
	}, nil
}

// ValidarXMLBytes valida um XML de NF-e a partir de bytes na memória
//
// Útil quando você já tem o XML carregado em memória ou de uma API
//
// Exemplo:
//
//	xmlData := []byte("<nfeProc>...</nfeProc>")
//	result, err := client.ValidarXMLBytes(xmlData, "schemas/v4/procNFe_v4.00.xsd")
func (c *Client) ValidarXMLBytes(xmlData []byte, xsdPath string) (*ValidationResult, error) {
	// 1. Validar XSD
	if err := ValidateWithXSD(xmlData, xsdPath); err != nil {
		return &ValidationResult{
			ValidoXSD: false,
			Erro:      fmt.Errorf("falha na validação XSD: %w", err),
		}, nil
	}

	// 2. Parse do XML
	nfe, err := validation.ParseNFe(xmlData)
	if err != nil {
		return &ValidationResult{
			ValidoXSD: true,
			Erro:      fmt.Errorf("falha ao parsear XML: %w", err),
		}, nil
	}

	// Extrair chave
	chave := validation.ExtractChaveFromID(nfe.InfNFe.ID)
	if chave == "" {
		chave = nfe.InfNFe.ID
	}

	// 3. Consultar SEFAZ
	status, err := c.sefaz.ConsultaSituacaoNFe(chave)
	if err != nil {
		return &ValidationResult{
			ValidoXSD:   true,
			ChaveAcesso: chave,
			DadosNFe:    convertInternalNFeData(nfe),
			Erro:        fmt.Errorf("falha na consulta SEFAZ: %w", err),
		}, nil
	}

	return &ValidationResult{
		ValidoXSD:   true,
		ChaveAcesso: chave,
		Autorizado:  status.Autorizado,
		Status: StatusSefaz{
			Codigo:   status.Codigo,
			Mensagem: status.Mensagem,
		},
		DadosNFe: convertInternalNFeData(nfe),
	}, nil
}

// ValidarChave consulta a situação de uma NF-e apenas pela chave de acesso
//
// Não valida XSD nem faz parse do XML. Apenas consulta o status na SEFAZ.
//
// Parâmetros:
//   - chave: chave de acesso de 44 dígitos
//
// Exemplo:
//
//	result, err := client.ValidarChave("35250732409620000175550010000037471011544648")
//	if result.Autorizado {
//	    fmt.Println("NF-e está autorizada!")
//	}
func (c *Client) ValidarChave(chave string) (*ValidationResult, error) {
	// Validar formato
	chaveClean := validation.OnlyDigits(chave)
	if len(chaveClean) != 44 {
		return nil, fmt.Errorf("chave de acesso inválida: deve ter 44 dígitos")
	}

	status, err := c.sefaz.ConsultaSituacaoNFe(chave)
	if err != nil {
		return &ValidationResult{
			ChaveAcesso: chave,
			Erro:        fmt.Errorf("falha na consulta SEFAZ: %w", err),
		}, nil
	}

	return &ValidationResult{
		ChaveAcesso: chave,
		ValidoXSD:   false, // N/A neste modo
		Autorizado:  status.Autorizado,
		Status: StatusSefaz{
			Codigo:   status.Codigo,
			Mensagem: status.Mensagem,
		},
	}, nil
}

// convertInternalNFeData converte a struct interna validation.NFeEnvelope para DadosNFe público
func convertInternalNFeData(nfe *validation.NFeEnvelope) *DadosNFe {
	return &DadosNFe{
		Modelo: nfe.InfNFe.Ide.Modelo,
		Serie:  nfe.InfNFe.Ide.Serie,
		Numero: nfe.InfNFe.Ide.NumNf,
		Emitente: Empresa{
			Documento: nfe.InfNFe.Emit.CNPJ,
			Nome:      nfe.InfNFe.Emit.XNome,
		},
		Destinatario: Empresa{
			Documento: validation.ChooseFirstNonEmpty(nfe.InfNFe.Dest.CNPJ, nfe.InfNFe.Dest.CPF),
			Nome:      nfe.InfNFe.Dest.XNome,
		},
		ValorTotal: nfe.InfNFe.Total.ICMSTot.VNF,
	}
}