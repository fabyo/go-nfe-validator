# Go NFE Validator 📄✅
<p>
   <img src="https://img.shields.io/badge/Go-1.24.0-00ADD8?style=for-the-badge&logo=go&logoColor=white" />
   <img src="https://img.shields.io/badge/XML%2FXSD-NFe-blueviolet?style=for-the-badge" alt="XML/XSD"/>   
   <img src="https://img.shields.io/badge/XML-XSD%20Schemas-orange?style=for-the-badge" />
   <img src="https://img.shields.io/badge/SEFAZ-4CAF50?style=for-the-badge" />
   <img src="https://img.shields.io/badge/License-MIT-4CAF50?style=for-the-badge&logo=opensourceinitiative&logoColor=white" />   
</p>


<img src="go-nfe.png" alt="Golang" width="200" />

Validador de **NF-e em Go**, focado em:

- ✅ **Validação XSD** usando *libxml2* via `go-xsd-validate`
- ✅ **Validação estrutural / de dados** (parse do XML)
- ✅ **Consulta real na SEFAZ** para verificar o status da NF
- ✅ Retorno em **JSON estruturado**, pronto para APIs, antifraude, auditoria etc.

---

## 🧠 O que o projeto faz

Dado um arquivo XML de NF-e ou procNFe (`<NFe>` ou `<nfeProc>`), o validador:

1. **Valida o XML contra o XSD oficial da NF-e 4.00**
2. Se o XSD passou, faz **parse do XML** e extrai:
   - **Modelo** (`mod`)
   - **Série** (`serie`)
   - **Número** (`nNF`)
   - **Chave de acesso** (`chNFe`)
   - **CNPJ** e razão social do emitente
   - **CNPJ/CPF** e nome do destinatário
   - **Valor total da nota** (`vNF`)
3. Opcionalmente, consulta a **SEFAZ real** para:
   - verificar se a nota existe,
   - se está **autorizada**, **cancelada**, **denegada**, etc.
4. Retorna um **JSON** com o resultado consolidado.

---

## 🧪 Exemplo de saída (JSON)

```json
{
  "tipo": "nfe",
  "chave_acesso": "12349874111111000123550010000040421000040420",
  "valido_xsd": true,
  "sefaz": {
    "consultado": true,
    "autorizado": true,
    "codigo": "100",
    "mensagem": "Autorizado o uso da NF-e"
  },
  "dados_xml": {
    "modelo": "55",
    "serie": "1",
    "numero": "4042",
    "emitente_cnpj": "12345678000199",
    "emitente_razao": "EMPRESA EXEMPLO LTDA",
    "destinatario_doc": "53745432000188",
    "destinatario_nome": "CLIENTE TESTE",
    "valor_total_nota": "199.90"
  }
}
```

---

## Uso como Biblioteca

```go
go get github.com/fabyo/go-nfe-validator
```

### 1️⃣ Validar apenas XSD (desenvolvimento)

```go
import "github.com/fabyo/go-nfe-validator/pkg/nfe"

xmlData, _ := os.ReadFile("nota.xml")
err := nfe.ValidarApenasXSD(xmlData, "schemas/v4/procNFe_v4.00.xsd")
```

### 2️⃣ Validar com SEFAZ
```go
client, _ := nfe.NewClient("cert", "key.pem", "cert.pem")
result, _ := client.ValidarXML(xmlData, "schema.xsd")

if result.Autorizado {
    fmt.Println("NF-e autorizada!")
}
```

### 3️⃣ Validar apenas por chave
```go
result, _ := client.ValidarChave("123456789098765433215550010000098765543211111")
fmt.Println(result.Status.Mensagem)
```

### 4️⃣ Script de exemplo
```go
go run examples/validar-xml/main.go 12345678998765432111111122222233333344444455-procNFe.xml
```

### 🚀 Outros projetos poderão usar assim:
```go
package main

import (
    "fmt"
    "os"
    
    "github.com/fabyo/go-nfe-validator/pkg/nfe"
)

func main() {
    // Validação rápida apenas XSD
    xmlData, _ := os.ReadFile("nota.xml")
    if err := nfe.ValidarApenasXSD(xmlData, "schema.xsd"); err != nil {
        panic(err)
    }
    
    // Ou validação completa
    client, _ := nfe.NewClient("cert", "key.pem", "cert.pem")
    result, _ := client.ValidarChave("12345678998765432111111122222233333344444455")
    
    fmt.Printf("Autorizada: %v\n", result.Autorizado)
}
```

---
## Uso como CLI:

1️⃣ **Apenas XSD (desenvolvimento - super rápido!)**
```bash
./validator -xsd nota.xml schemas/v4/procNFe_v4.00.xsd
```
✅ Valida apenas se o XML está correto conforme o schema  
✅ Perfeito para desenvolvimento de emissor  
✅ Não consulta SEFAZ  
✅ Resposta instantânea  

2️⃣ **XSD + Parse (validação intermediária)**
```bash
./validator -skip-sefaz nota.xml schemas/v4/procNFe_v4.00.xsd
```
✅ Valida XSD  
✅ Extrai e valida dados (chave, CNPJ, valores)  
✅ Não consulta SEFAZ  
✅ Bom para testes antes de ir para SEFAZ  

3️⃣ **Validação Completa (produção)**
```bash
./validator nota.xml schemas/v4/procNFe_v4.00.xsd
```
✅ Valida XSD  
✅ Valida dados  
✅ Consulta status na SEFAZ  
✅ Retorna status da nota

4️⃣ **Validação pela chave (sem xml)**
```bash
./validator -chave=12345678912345678998765432112345678911111111
```
✅ Verifica se tem exatamente 44 dígitos  
✅ Verifica se são apenas números  
✅ Consulta status na SEFAZ  
✅ Retorna erro claro se inválida  
✅ Retorna status da nota  

<img src="status.png" alt="Golang" width="700" />

---

# Exemplo: .env.production
```bash
# export NFE_ENV=production
NFE_ENV=production

# -----------------
# Certificados
# -----------------
NFE_CERT_DIR=certs/
NFE_CERT_KEY_FILE=key.pem
NFE_CERT_PUB_FILE=cert.pem

# -----------------
# Credenciais
# -----------------
NFE_CNPJ=12345678000100
NFE_UF_IBGE=35

# -----------------
# URLs (Produção)
# -----------------
SEFAZ_CONSULTA_URL=https://nfe.fazenda.sp.gov.br/ws/nfeconsultaprotocolo4.asmx
```

---

## 🧩 Fluxo Inteligente

Fluxo lógico atual do validador:

```mermaid
graph TD
    %% Fluxo principal da NFe
    A[Valida XSD]
    B[Erro de schema]
    C[Parse XML]
    D[XML inválido]
    E[Consulta SEFAZ]
    F[Retorna apenas dados do XML]
    G[Status real da NFe]

    A -- erro --> B
    A -- ok --> C
    C -- erro --> D
    C -- ok --> E
    E -- "skip sefaz" --> F
    E -- consulta --> G

    %% Fluxo separado: validação por chave de acesso
    X[Valida chave de acesso]
    Y[Chave inválida]
    Z[Chave válida]

    X -- inválida --> Y
    X -- válida --> Z

```

Em resumo:

- **XSD sempre roda primeiro**.
- Se o XSD falhar → erro e fim.
- Se o XSD passar:
  - faz parse do XML (para extrair dados de nota);
  - se não estiver em modo “só XSD” e não usar `--skip-sefaz`, consulta a SEFAZ e enriquece o resultado com o status real da NF-e.

---

## 📚 Schemas (XSD) via `sefaz-scraper`

Os schemas oficiais **não ficam hardcoded**:  
este projeto usa os XSDs atualizados pelo:

- 🔗 [`fabyo/sefaz-scraper`](https://github.com/fabyo/sefaz-scraper)

```bash
./download_schemas.sh
```

A ideia é:

- `sefaz-scraper` baixa/atualiza os XSDs direto das SEFAZ/Portal;
- `go-nfe-validator` aponta para essa pasta, garantindo validação sempre com os **layouts oficiais mais recentes**.

---

## 🎯 Objetivo do projeto

Ser um **núcleo técnico** sólido para:

- validação de NF-e (estrutura + XSD),
- conferência real na SEFAZ,
- saída estruturada em JSON,
- base para:
  - antifraude,
  - robôs de conferência fiscal,
  - integrações com outros sistemas (ERPs, BI, IA, etc.).
