package main

import (
	"fmt"
	"log"
	"os"

	"github.com/fabyo/go-nfe-validator/pkg/nfe"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Uso: go run main.go <arquivo.xml>")
		os.Exit(1)
	}

	xmlPath := os.Args[1]

	// Detectar caminho do schema automaticamente
	schemaPath := "schemas/v4/procNFe_v4.00.xsd"
	if _, err := os.Stat(schemaPath); os.IsNotExist(err) {
		// Tentar caminho relativo da raiz do projeto
		schemaPath = "../../schemas/v4/procNFe_v4.00.xsd"
	}

	// 1. Validar apenas XSD (rápido)
	fmt.Println("🔍 Validando XSD...")
	xmlData, err := os.ReadFile(xmlPath)
	if err != nil {
		log.Fatal(err)
	}

	if err := nfe.ValidarApenasXSD(xmlData, schemaPath); err != nil {
		log.Fatal("❌ XML inválido:", err)
	}
	fmt.Println("✅ XSD válido")

	// 2. Parse dos dados
	fmt.Println("\n📄 Extraindo dados...")
	dados, err := nfe.ParsearXML(xmlData)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Modelo: %s\n", dados.Modelo)
	fmt.Printf("Série: %s / Número: %s\n", dados.Serie, dados.Numero)
	fmt.Printf("Emitente: %s (%s)\n", dados.Emitente.Nome, dados.Emitente.Documento)
	fmt.Printf("Destinatário: %s (%s)\n", dados.Destinatario.Nome, dados.Destinatario.Documento)
	fmt.Printf("Valor Total: R$ %s\n", dados.ValorTotal)

	// 3. Validar com SEFAZ (opcional)
	fmt.Println("\n🌐 Consultando SEFAZ...")
	client, err := nfe.NewClientFromEnv()
	if err != nil {
		fmt.Println("⚠️ Não foi possível criar cliente SEFAZ (configure .env.production)")
		return
	}

	result, err := client.ValidarXMLBytes(xmlData, schemaPath)
	if err != nil {
		log.Fatal(err)
	}

	if result.Erro != nil {
		fmt.Printf("⚠️ Erro: %v\n", result.Erro)
		return
	}

	if result.Status.IsAutorizado() {
		fmt.Println("✅ NF-e AUTORIZADA pela SEFAZ")
	} else if result.Status.IsCancelado() {
		fmt.Println("❌ NF-e CANCELADA")
	} else {
		fmt.Printf("Status: %s - %s\n", result.Status.Codigo, result.Status.Mensagem)
	}
}
