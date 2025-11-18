package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/fabyo/go-nfe-validator/internal/config"
	"github.com/fabyo/go-nfe-validator/internal/sefaz"
	"github.com/fabyo/go-nfe-validator/internal/validation"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("⚡️ Iniciando Validador NF-e")

	// --- FLAGS DE LINHA DE COMANDO ---
	xsdOnly := flag.Bool("xsd", false, "Validar apenas contra XSD (sem consulta SEFAZ)")
	skipSefaz := flag.Bool("skip-sefaz", false, "Pular consulta SEFAZ (valida XSD + parse dados)")
	chaveAcesso := flag.String("chave", "", "Consultar apenas pela chave de acesso (44 dígitos)")
	
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Uso: %s [opções] <arquivo_xml> <arquivo_xsd>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "   ou: %s -chave=<44_digitos>\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Opções:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nExemplos:")
		fmt.Fprintln(os.Stderr, "  # Validação completa (XSD + Parse + SEFAZ)")
		fmt.Fprintln(os.Stderr, "  ./validator nota.xml schema.xsd")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  # Apenas validação XSD (desenvolvimento)")
		fmt.Fprintln(os.Stderr, "  ./validator -xsd nota.xml schema.xsd")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  # XSD + Parse, sem consultar SEFAZ")
		fmt.Fprintln(os.Stderr, "  ./validator -skip-sefaz nota.xml schema.xsd")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "  # Consulta direta por chave de acesso (sem XML)")
		fmt.Fprintln(os.Stderr, "  ./validator -chave=35250732409620000175550010000037471011544648")
	}
	
	flag.Parse()

	// --- MODO: CONSULTA APENAS POR CHAVE ---
	if *chaveAcesso != "" {
		validateByChave(*chaveAcesso)
		return
	}

	// Validar argumentos para modo normal
	if flag.NArg() < 2 {
		flag.Usage()
		os.Exit(1)
	}

	xmlPath := flag.Arg(0)
	xsdPath := flag.Arg(1)

	// Carregar configuração
	cfg := config.Load()
	
	log.Printf("Ambiente ativo: %s (UF %s)", cfg.Env, cfg.UF)
	
	if *xsdOnly {
		log.Println("Nível de validação: XSD apenas")
	} else if *skipSefaz {
		log.Println("Nível de validação: XSD + Parse")
	} else {
		log.Println("Nível de validação: Completa (XSD + Parse + SEFAZ)")
	}

	// Resultado da validação
	result := validation.ValidationResponse{
		Tipo: "nfe",
	}

	// --- FASE 1: VALIDAÇÃO XSD (SEMPRE OBRIGATÓRIA) ---
	log.Println("➡️ Fase 1: Validação XSD...")
	
	xmlData, err := os.ReadFile(xmlPath)
	if err != nil {
		result.ValidoXSD = false
		result.Erro = fmt.Sprintf("Erro ao ler arquivo XML: %v", err)
		printResult(result)
		os.Exit(1)
	}
	
	if err := validation.ValidateWithXSD(xmlData, xsdPath); err != nil {
		result.ValidoXSD = false
		result.Erro = fmt.Sprintf("Falha na validação XSD: %v", err)
		printResult(result)
		os.Exit(1)
	}
	result.ValidoXSD = true
	log.Println("   ✅ XSD válido")

	// Se apenas XSD, retornar aqui
	if *xsdOnly {
		log.Println("✅ Validação XSD concluída. Pulando fases 2 e 3 (--xsd ativo)")
		printResult(result)
		return
	}

	// --- FASE 2: PARSE DO XML ---
	log.Println("➡️ Fase 2: Parse do XML...")
	nfe, err := validation.ParseNFe(xmlData)
	if err != nil {
		result.Erro = fmt.Sprintf("Falha ao parsear XML: %v", err)
		printResult(result)
		os.Exit(1)
	}

	// Extrair chave de acesso
	result.ChaveAcesso = validation.ExtractChaveFromID(nfe.InfNFe.ID)
	if result.ChaveAcesso == "" {
		result.ChaveAcesso = nfe.InfNFe.ID
	}

	// Preencher dados do XML
	result.DadosXML = &validation.DadosXMLNFe{
		Modelo:       nfe.InfNFe.Ide.Modelo,
		Serie:        nfe.InfNFe.Ide.Serie,
		Numero:       nfe.InfNFe.Ide.NumNf,
		EmitCNPJ:     nfe.InfNFe.Emit.CNPJ,
		EmitRazao:    nfe.InfNFe.Emit.XNome,
		DestDoc:      validation.ChooseFirstNonEmpty(nfe.InfNFe.Dest.CNPJ, nfe.InfNFe.Dest.CPF),
		DestNome:     nfe.InfNFe.Dest.XNome,
		ValorTotalNF: nfe.InfNFe.Total.ICMSTot.VNF,
	}
	log.Println("   ✅ XML parseado com sucesso")

	// Se skip-sefaz, retornar aqui
	if *skipSefaz {
		log.Println("✅ Validação XSD + Parse concluída. Pulando fase 3 (--skip-sefaz ativo)")
		result.Sefaz = validation.SefazStatus{
			Autorizado: false,
			Codigo:     "N/A",
			Mensagem:   "Consulta SEFAZ não realizada (--skip-sefaz)",
		}
		printResult(result)
		return
	}

	// --- FASE 3: CONSULTA SEFAZ ---
	log.Println("➡️ Fase 3: Consulta SEFAZ (mTLS)...")

	client, err := sefaz.NewClient(cfg)
	if err != nil {
		result.Erro = fmt.Sprintf("Falha ao configurar cliente SEFAZ: %v", err)
		printResult(result)
		os.Exit(1)
	}

	status, err := client.ConsultaSituacaoNFe(result.ChaveAcesso)
	if err != nil {
		result.Erro = fmt.Sprintf("Falha na consulta remota: %v", err)
		result.Sefaz = validation.SefazStatus{
			Autorizado: false,
			Codigo:     "",
			Mensagem:   "",
		}
		printResult(result)
		os.Exit(1)
	}

	result.Sefaz = status
	log.Printf("✅ FINAL: Status %s - %s", status.Codigo, status.Mensagem)

	printResult(result)
}

// printResult imprime o resultado em JSON
func printResult(result validation.ValidationResponse) {
	jsonOutput, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		log.Fatalf("❌ Erro ao gerar JSON: %v", err)
	}
	fmt.Println(string(jsonOutput))
}

// validateByChave consulta SEFAZ apenas com a chave de acesso (sem XML)
func validateByChave(chave string) {
	log.Println("🔑 Modo: Consulta por chave de acesso")
	
	// Validar formato da chave (44 dígitos)
	if len(chave) != 44 {
		log.Fatalf("❌ Chave de acesso inválida. Deve ter exatamente 44 dígitos. Recebido: %d dígitos", len(chave))
	}

	// Verificar se são todos números
	chaveClean := validation.OnlyDigits(chave)
	if len(chaveClean) != 44 {
		log.Fatalf("❌ Chave de acesso inválida. Deve conter apenas números.")
	}

	log.Printf("Chave: %s", chave)

	// Carregar configuração
	cfg := config.Load()
	log.Printf("Ambiente: %s (UF %s)", cfg.Env, cfg.UF)

	// Configurar cliente SEFAZ
	client, err := sefaz.NewClient(cfg)
	if err != nil {
		log.Fatalf("❌ Falha ao configurar cliente SEFAZ: %v", err)
	}

	log.Println("➡️ Consultando SEFAZ...")

	status, err := client.ConsultaSituacaoNFe(chave)
	
	result := validation.ValidationResponse{
		Tipo:        "nfe",
		ChaveAcesso: chave,
		ValidoXSD:   false,
	}
	
	if err != nil {
		result.Sefaz = validation.SefazStatus{
			Autorizado: false,
			Codigo:     "999",
			Mensagem:   "Erro na consulta",
		}
		result.Erro = fmt.Sprintf("Falha na consulta: %v", err)
		printResult(result)
		os.Exit(1)
	}

	log.Printf("✅ Status %s - %s", status.Codigo, status.Mensagem)

	result.Sefaz = status
	printResult(result)
}
