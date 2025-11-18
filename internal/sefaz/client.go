package sefaz

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/fabyo/go-nfe-validator/internal/config"
	"github.com/fabyo/go-nfe-validator/internal/validation"
)

// Regex para extrair cStat e xMotivo da resposta XML da SEFAZ
var cStatRegex = regexp.MustCompile(`<cStat>(\d+)</cStat>`)
var xMotivoRegex = regexp.MustCompile(`<xMotivo>(.*?)</xMotivo>`)

// --- CLIENT STRUCT ---
type Client struct {
	http *http.Client
	cfg  *config.Config
}

// --- Funções Auxiliares (CA Loading) ---

// loadCertsFromDir: Carrega todos os certificados .crt e .pem de um diretório e os adiciona ao pool.
func loadCertsFromDir(pool *x509.CertPool, dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("falha ao ler o diretório %s: %w", dir, err)
	}

	for _, entry := range entries {
		name := entry.Name()
		
		// Pular arquivos que não são certificados CA
		if entry.IsDir() || strings.Contains(name, "key.pem") {
			continue
		}
		
		// Carregar apenas .crt e .pem (exceto key.pem)
		if strings.HasSuffix(name, ".crt") || strings.HasSuffix(name, ".pem") {
			path := filepath.Join(dir, name)
			certBytes, err := os.ReadFile(path)
			if err != nil {
				log.Printf("⚠️ Aviso: Falha ao ler arquivo %s: %v", path, err)
				continue
			}
			if ok := pool.AppendCertsFromPEM(certBytes); !ok {
				log.Printf("⚠️ Aviso: Falha ao adicionar CA do arquivo %s (formato inválido).", name)
			}
		}
	}
	return nil
}

// --- CONSTRUTOR ---
// NewClient: Configura o cliente HTTP com o certificado mTLS necessário
func NewClient(cfg *config.Config) (*Client, error) {
	// Caminhos completos dos arquivos do certificado de cliente
	keyPath := filepath.Join(cfg.CertDir, cfg.CertKeyFile)
	certPath := filepath.Join(cfg.CertDir, cfg.CertPubFile)

	// 1. Carregar Chaves e Certificado do Cliente
	cert, err := tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("falha ao carregar chaves PEM (%s/%s): %w", cfg.CertDir, cfg.CertPubFile, err)
	}

	// 2. Configurar Pool de Confiança (RootCAs)
	caCertPool, err := x509.SystemCertPool()
	if err != nil || caCertPool == nil {
		log.Println("⚠️ Aviso: SystemCertPool falhou ou retornou nil. Usando pool vazio.")
		caCertPool = x509.NewCertPool()
	}

	// 3. Carregar CAs do ICP-Brasil (Resolve o erro de confiança no servidor)
	if err := loadCertsFromDir(caCertPool, cfg.CertDir); err != nil {
		return nil, fmt.Errorf("erro ao carregar CAs da pasta %s: %w", cfg.CertDir, err)
	}

	// 4. Configurações mTLS e Protocolo
	// ⚡ CORREÇÃO CRÍTICA: Habilitar renegociação TLS (exigido pela SEFAZ SP e Nacional)
	tlsConfig := &tls.Config{
		Certificates:  []tls.Certificate{cert},
		RootCAs:       caCertPool,
		Renegotiation: tls.RenegotiateFreelyAsClient, // ← MUDANÇA AQUI!
		MinVersion:    tls.VersionTLS12,
		MaxVersion:    tls.VersionTLS12,
	}

	httpClient := &http.Client{
		Timeout: 15 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
			Proxy:           http.ProxyFromEnvironment,
			MaxIdleConns:    10,
			IdleConnTimeout: 30 * time.Second,
		},
	}

	return &Client{http: httpClient, cfg: cfg}, nil
}

// --- MÉTODO DE NEGÓCIO ---
// ConsultaSituacaoNFe: Consulta a situação da NF-e no SEFAZ (Webservice NfeConsultaNFe4)
func (c *Client) ConsultaSituacaoNFe(chaveAcesso string) (validation.SefazStatus, error) {
	
	soapAction := "http://www.portalfiscal.inf.br/nfe/wsdl/NfeConsultaNFe4/nfeConsultaNF"
	sefazUrl := c.cfg.ConsultaURL 

	// O XML de Consulta de Situação (sem quebras de linha - SEFAZ SP é sensível!)
	soapEnv := fmt.Sprintf(`<soap12:Envelope xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xmlns:xsd="http://www.w3.org/2001/XMLSchema" xmlns:soap12="http://www.w3.org/2003/05/soap-envelope"><soap12:Body><nfeDadosMsg xmlns="http://www.portalfiscal.inf.br/nfe/wsdl/NFeConsultaProtocolo4"><consSitNFe xmlns="http://www.portalfiscal.inf.br/nfe" versao="4.00"><tpAmb>1</tpAmb><xServ>CONSULTAR</xServ><chNFe>%s</chNFe></consSitNFe></nfeDadosMsg></soap12:Body></soap12:Envelope>`, chaveAcesso)

	req, err := http.NewRequest("POST", sefazUrl, strings.NewReader(soapEnv))
	if err != nil {
		return validation.SefazStatus{Codigo: "999"}, fmt.Errorf("erro ao criar requisição: %w", err)
	}

	req.Header.Set("Content-Type", `application/soap+xml; charset=utf-8; action="`+soapAction+`"`)

	resp, err := c.http.Do(req)
	if err != nil {
		return validation.SefazStatus{Codigo: "999"}, fmt.Errorf("erro na conexão mTLS/webservice: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return validation.SefazStatus{Codigo: "999"}, fmt.Errorf("erro ao ler resposta: %w", err)
	}

	// DEBUG: Ver a resposta completa da SEFAZ
	log.Printf("📄 Resposta SEFAZ:\n%s", string(body))

	// Analisa a resposta XML...
	bodyStr := string(body)
	cStatMatch := cStatRegex.FindStringSubmatch(bodyStr)
	xMotivoMatch := xMotivoRegex.FindStringSubmatch(bodyStr)

	cStat := "999"
	xMotivo := "Resposta da SEFAZ não parseada."

	if len(cStatMatch) > 1 {
		cStat = cStatMatch[1]
	}

	if len(xMotivoMatch) > 1 {
		xMotivo = xMotivoMatch[1]
	} else if strings.Contains(bodyStr, "<xMotivo>") {
		// Tenta extrair o motivo de forma mais robusta caso o regex falhe
		parts := strings.Split(bodyStr, "<xMotivo>")
		if len(parts) > 1 {
			xMotivo = strings.Split(parts[1], "</xMotivo>")[0]
		}
	}

	status := validation.SefazStatus{
		Codigo:   cStat,
		Mensagem: xMotivo,
	}

	// Status 100 (Autorizada) ou 110 (Em processamento, mas autorizado)
	if cStat == "100" || cStat == "110" {
		status.Autorizado = true
	} else if cStat == "101" {
		// 101: Cancelamento de NF-e Homologado
		status.Autorizado = false
	} else {
		status.Autorizado = false
	}

	return status, nil
}