package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Env          string
	CertDir      string
	CertKeyFile  string
	CertPubFile  string
	CNPJ         string
	UF           string
	ConsultaURL  string
	DistURL      string
}

// Load carregar a configuração com base na variável NFE_ENV ou padroniza para 'production'.
func Load() *Config {
	// Pega NFE_ENV do ambiente global para decidir qual arquivo carregar
	env := os.Getenv("NFE_ENV")
	if env == "" {
		env = "production"
	}
	
	// Cria o nome do arquivo (ex: .env.production)
	envFile := fmt.Sprintf(".env.%s", env)
	
	// Carrega o arquivo .env apropriado
	if err := godotenv.Load(envFile); err != nil {
		// É comum que o erro ocorra se o .env principal não existir;
		// verificamos explicitamente o erro de arquivo não encontrado
		if !strings.Contains(err.Error(), "no such file or directory") {
			log.Fatalf("Erro ao carregar arquivo de ambiente %s: %v", envFile, err)
		} else {
            // Se o arquivo não existe, apenas avisa e segue usando variáveis de ambiente do sistema.
            log.Printf("Aviso: Arquivo de ambiente '%s' não encontrado. Usando variáveis de ambiente do sistema.", envFile)
        }
	}

	return &Config{
		Env:          env,
		CertDir:      os.Getenv("NFE_CERT_DIR"),
		CertKeyFile:  os.Getenv("NFE_CERT_KEY_FILE"),
		CertPubFile:  os.Getenv("NFE_CERT_PUB_FILE"),
		CNPJ:         os.Getenv("NFE_CNPJ"),
		UF:           os.Getenv("NFE_UF_IBGE"),
		ConsultaURL:  os.Getenv("SEFAZ_CONSULTA_URL"),
		DistURL:      os.Getenv("SEFAZ_DIST_URL"),
	}
}