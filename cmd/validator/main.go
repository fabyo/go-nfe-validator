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

// --- FUNÇÃO AUXILIAR: Saída JSON Única ---
func printJSONAndExit(v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		// Fallback em caso de erro gravíssimo no JSON
		fmt.Println(`{"erro": "falha interna ao gerar JSON"}`)
		os.Exit(1)
	}
	fmt.Println(string(b))
	os.Exit(0)
}

func main() {
	// Remove prefixos de log para não sujar o output antes do JSON final
	log.SetFlags(0)

	// 1. DEFINIÇÃO DAS FLAGS (CLI)
	xsdOnly := flag.Bool("xsd", false, "Modo 1: Apenas validação XSD (super rápido, offline)")
	skipSefaz := flag.Bool("skip-sefaz", false, "Modo 2: Valida XSD + Parse, mas pula consulta SEFAZ")
	envFlag := flag.String("env", "production", "Ambiente: 'production' ou 'homolog'")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Uso: %s [flags] <arquivo_xml> <arquivo_xsd>\n\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "Opções:")
		flag.PrintDefaults()
		fmt.Fprintln(os.Stderr, "\nExemplos:")
		fmt.Fprintln(os.Stderr, "  ./validator -xsd nota.xml schema.xsd")
		fmt.Fprintln(os.Stderr, "  ./validator -skip-sefaz -env=homolog nota.xml schema.xsd")
		fmt.Fprintln(os.Stderr, "  ./validator nota.xml schema.xsd")
	}

	flag.Parse()

	// Validação de Argumentos Posicionais (XML e XSD)
	args := flag.Args()
	if len(args) < 2 {
		log.Println("❌ ERRO: Caminhos do XML e XSD são obrigatórios.")
		flag.Usage()
		os.Exit(1)
	}

	xmlPath := args[0]
	xsdPath := args[1]

	// 2. LEITURA DO ARQUIVO XML (Necessária para todas as fases)
	xmlBytes, err := os.ReadFile(xmlPath)
	if err != nil {
		printJSONAndExit(validation.ValidationResponse{
			Tipo: "nfe",
			Erro: fmt.Sprintf("Erro ao ler arquivo XML: %v", err),
		})
	}

	// ============================================================
	// FASE 1: VALIDAÇÃO XSD (Roda em todos os modos)
	// ============================================================
	// log.Println("➡️ Fase 1: Validação XSD...") // (Comentado para manter output limpo)

	if err := validation.ValidateWithXSD(xmlBytes, xsdPath); err != nil {
		printJSONAndExit(validation.ValidationResponse{
			Tipo:      "nfe",
			ValidoXSD: false,
			Erro:      fmt.Sprintf("XML inválido contra XSD: %v", err),
		})
	}

	// 🛑 PONTO DE SAÍDA: MODO 1 (-xsd)
	if *xsdOnly {
		printJSONAndExit(validation.ValidationResponse{
			Tipo:      "nfe",
			ValidoXSD: true,
			// Campos opcionais omitidos para resposta curta
		})
	}

	// ============================================================
	// FASE 2: PARSE E EXTRAÇÃO DE DADOS (Modos 2 e 3)
	// ============================================================
	
	nfeEnv, err := validation.ParseNFe(xmlBytes)
	if err != nil {
		printJSONAndExit(validation.ValidationResponse{
			Tipo:      "nfe",
			ValidoXSD: true, // Passou no XSD, mas falhou no Parse (estranho, mas possível)
			Erro:      fmt.Sprintf("Erro ao estruturar NFe (Parse): %v", err),
		})
	}

	chave := validation.ExtractChaveFromID(nfeEnv.InfNFe.ID)
	if chave == "" {
		printJSONAndExit(validation.ValidationResponse{
			Tipo:      "nfe",
			ValidoXSD: true,
			Erro:      "Chave de acesso não encontrada no XML (infNFe/@Id)",
		})
	}

	// Monta o objeto de dados extraídos
	// (Corrigido para usar os campos reais das structs que definimos em internal/validation)
	dados := &validation.DadosXMLNFe{
		Modelo:       nfeEnv.InfNFe.Ide.Modelo,
		Serie:        nfeEnv.InfNFe.Ide.Serie,
		Numero:       nfeEnv.InfNFe.Ide.NumNf,
		EmitCNPJ:     validation.OnlyDigits(nfeEnv.InfNFe.Emit.CNPJ),
		EmitRazao:    nfeEnv.InfNFe.Emit.XNome,
		DestDoc:      validation.ChooseFirstNonEmpty(validation.OnlyDigits(nfeEnv.InfNFe.Dest.CNPJ), validation.OnlyDigits(nfeEnv.InfNFe.Dest.CPF)),
		DestNome:     nfeEnv.InfNFe.Dest.XNome,
		ValorTotalNF: nfeEnv.InfNFe.Total.ICMSTot.VNF,
	}

	// 🛑 PONTO DE SAÍDA: MODO 2 (-skip-sefaz)
	if *skipSefaz {
		printJSONAndExit(validation.ValidationResponse{
			Tipo:        "nfe",
			ChaveAcesso: chave,
			ValidoXSD:   true,
			Sefaz: validation.SefazStatus{
				Codigo:   "N/A",
				Mensagem: "Consulta SEFAZ não realizada (--skip-sefaz)",
				Autorizado: false,
			},
			DadosXML: dados,
		})
	}

	// ============================================================
	// FASE 3: CONSULTA SEFAZ (Modo Padrão / Produção)
	// ============================================================

	// Configura o ambiente para o config.Load() ler
	os.Setenv("NFE_ENV", *envFlag)
	cfg := config.Load()

	// Inicializa Cliente mTLS
	client, err := sefaz.NewClient(cfg)
	if err != nil {
		printJSONAndExit(validation.ValidationResponse{
			Tipo:        "nfe",
			ChaveAcesso: chave,
			ValidoXSD:   true,
			DadosXML:    dados,
			Erro:        fmt.Sprintf("Falha na configuração mTLS (Certificados): %v", err),
		})
	}

	// Executa Consulta
	sefazStatus, err := client.ConsultaSituacaoNFe(chave)
	if err != nil {
		printJSONAndExit(validation.ValidationResponse{
			Tipo:        "nfe",
			ChaveAcesso: chave,
			ValidoXSD:   true,
			DadosXML:    dados,
			Erro:        fmt.Sprintf("Falha na comunicação com a SEFAZ: %v", err),
		})
	}

	// Monta a resposta final completa
	resp := validation.ValidationResponse{
		Tipo:        "nfe",
		ChaveAcesso: chave,
		ValidoXSD:   true,
		Sefaz:       sefazStatus,
		DadosXML:    dados,
	}

	// Sucesso total!
	printJSONAndExit(resp)
}