package nfe_test

import (
	"fmt"
	"log"
	"os"

	"github.com/fabyo/go-nfe-validator/pkg/nfe"
)

// Exemplo básico: validar apenas XSD (desenvolvimento)
func ExampleValidarApenasXSD() {
	xmlData, err := os.ReadFile("testdata/nota.xml")
	if err != nil {
		log.Fatal(err)
	}

	err = nfe.ValidarApenasXSD(xmlData, "schemas/v4/procNFe_v4.00.xsd")
	if err != nil {
		fmt.Println("XML inválido:", err)
		return
	}

	fmt.Println("XML válido!")
}

// Exemplo: fazer parse do XML sem validar
func ExampleParsearXML() {
	xmlData, err := os.ReadFile("testdata/nota.xml")
	if err != nil {
		log.Fatal(err)
	}

	dados, err := nfe.ParsearXML(xmlData)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Emitente: %s\n", dados.Emitente.Nome)
	fmt.Printf("Valor: R$ %s\n", dados.ValorTotal)
}

// Exemplo: criar cliente e validar XML completo
func ExampleClient_ValidarXML() {
	// Criar cliente
	client, err := nfe.NewClient(nfe.Config{
		CertDir:     "cert",
		CertKeyFile: "key.pem",
		CertPubFile: "cert.pem",
		UF:          "35",
		Env:         "production",
	})
	if err != nil {
		log.Fatal(err)
	}

	// Validar XML completo (XSD + Parse + SEFAZ)
	result, err := client.ValidarXML("testdata/nota.xml", "schemas/v4/procNFe_v4.00.xsd")
	if err != nil {
		log.Fatal(err)
	}

	// Verificar resultado
	if result.Erro != nil {
		fmt.Println("Erro na validação:", result.Erro)
		return
	}

	if result.Autorizado {
		fmt.Println("✅ NF-e autorizada!")
		fmt.Printf("Status: %s - %s\n", result.Status.Codigo, result.Status.Mensagem)
	} else {
		fmt.Println("❌ NF-e não autorizada")
		fmt.Printf("Status: %s - %s\n", result.Status.Codigo, result.Status.Mensagem)
	}
}

// Exemplo: validar apenas por chave de acesso
func ExampleClient_ValidarChave() {
	// Criar cliente usando variáveis de ambiente
	client, err := nfe.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	// Validar apenas pela chave (sem XML)
	chave := "35250732409620000175550010000037471011544648"
	result, err := client.ValidarChave(chave)
	if err != nil {
		log.Fatal(err)
	}

	// Verificar usando métodos helper
	if result.Status.IsAutorizado() {
		fmt.Println("✅ NF-e autorizada")
	} else if result.Status.IsCancelado() {
		fmt.Println("❌ NF-e cancelada")
	} else if result.Status.IsDenegado() {
		fmt.Println("❌ NF-e denegada")
	} else {
		fmt.Printf("Status: %s\n", result.Status.Mensagem)
	}
}

// Exemplo: validar XML em bytes (útil para APIs)
func ExampleClient_ValidarXMLBytes() {
	client, err := nfe.NewClient(nfe.Config{
		CertDir:     "cert",
		CertKeyFile: "key.pem",
		CertPubFile: "cert.pem",
		UF:          "35",
		Env:         "production",
	})
	if err != nil {
		log.Fatal(err)
	}

	// XML recebido de uma API, por exemplo
	xmlData := []byte(`<nfeProc>...</nfeProc>`)

	result, err := client.ValidarXMLBytes(xmlData, "schemas/v4/procNFe_v4.00.xsd")
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Chave: %s\n", result.ChaveAcesso)
	fmt.Printf("Autorizado: %v\n", result.Autorizado)
}

// Exemplo: usar constantes de status
func ExampleStatusSefaz_IsAutorizado() {
	client, _ := nfe.NewClientFromEnv()
	result, _ := client.ValidarChave("35250732409620000175550010000037471011544648")

	// Usar os métodos helper
	switch {
	case result.Status.IsAutorizado():
		fmt.Println("NF-e autorizada e válida")
	case result.Status.IsCancelado():
		fmt.Println("NF-e foi cancelada")
	case result.Status.IsDenegado():
		fmt.Println("NF-e teve uso denegado")
	case result.Status.IsNaoEncontrado():
		fmt.Println("NF-e não existe na base da SEFAZ")
	default:
		fmt.Printf("Status: %s\n", result.Status.Mensagem)
	}
}

// Exemplo: fluxo completo de validação com tratamento de erros
func Example_fluxoCompleto() {
	// 1. Criar cliente
	client, err := nfe.NewClient(nfe.Config{
		CertDir:     "cert",
		CertKeyFile: "key.pem",
		CertPubFile: "cert.pem",
		UF:          "35",
		Env:         "production",
	})
	if err != nil {
		log.Fatal("Erro ao criar cliente:", err)
	}

	// 2. Ler XML
	xmlData, err := os.ReadFile("nota.xml")
	if err != nil {
		log.Fatal("Erro ao ler XML:", err)
	}

	// 3. Validar apenas XSD primeiro (rápido, sem consumir cota SEFAZ)
	if err := nfe.ValidarApenasXSD(xmlData, "schemas/v4/procNFe_v4.00.xsd"); err != nil {
		fmt.Println("❌ XML inválido (não passou no XSD):", err)
		return
	}
	fmt.Println("✅ XML válido (passou no XSD)")

	// 4. Fazer parse para ver os dados
	dados, err := nfe.ParsearXML(xmlData)
	if err != nil {
		log.Fatal("Erro ao parsear:", err)
	}
	fmt.Printf("📄 NF-e %s-%s de %s\n", dados.Serie, dados.Numero, dados.Emitente.Nome)

	// 5. Validar com SEFAZ
	result, err := client.ValidarXMLBytes(xmlData, "schemas/v4/procNFe_v4.00.xsd")
	if err != nil {
		log.Fatal("Erro na validação:", err)
	}

	if result.Erro != nil {
		fmt.Println("❌ Erro:", result.Erro)
		return
	}

	// 6. Verificar status
	if result.Autorizado {
		fmt.Println("✅ NF-e autorizada pela SEFAZ")
		fmt.Printf("   Protocolo: %s\n", result.Status.Mensagem)
	} else {
		fmt.Printf("❌ NF-e não autorizada: %s\n", result.Status.Mensagem)
	}
}