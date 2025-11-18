# Go NFE Validator 📄✅
<p>
   <img src="https://img.shields.io/badge/Go-1.24.0-00ADD8?style=for-the-badge&logo=go&logoColor=white" />
   <img src="https://img.shields.io/badge/XML%2FXSD-NFe-blueviolet?style=for-the-badge" alt="XML/XSD"/>   
  <img src="https://img.shields.io/badge/XML-XSD%20Schemas-orange?style=for-the-badge" />
  <img src="https://img.shields.io/badge/SEFAZ-4CAF50?style=for-the-badge" />
</p>

<img src="go-nfe.png" alt="Golang" width="200" />

Validador de **NF-e em Go**, focado em:

- ✅ **Validação XSD** usando *libxml2* via `go-xsd-validate`
- ✅ **Validação estrutural / de dados** (parse do XML)
- ✅ **Consulta real na SEFAZ** para verificar o status da NF
- ✅ Retorno em **JSON estruturado**, pronto para APIs, antifraude, auditoria etc.

---

## 🧠 O que o projeto faz hoje

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

## 🚀 Uso rápido

Exemplo de execução:

1️⃣ Apenas XSD (desenvolvimento - super rápido!)
```bash
./validator -xsd nota.xml schemas/v4/procNFe_v4.00.xsd
```
✅ Valida apenas se o XML está correto conforme o schema  
✅ Perfeito para desenvolvimento de emissor  
✅ Não consulta SEFAZ  
✅ Resposta instantânea  

2️⃣ XSD + Parse (validação intermediária)
```bash
./validator -skip-sefaz nota.xml schemas/v4/procNFe_v4.00.xsd
```
✅ Valida XSD  
✅ Extrai e valida dados (chave, CNPJ, valores)  
✅ Não consulta SEFAZ  
✅ Bom para testes antes de ir para SEFAZ  

3️⃣ Validação Completa (produção)
```bash
./validator nota.xml schemas/v4/procNFe_v4.00.xsd
```
✅ Valida XSD  
✅ Valida dados  
✅ Consulta status na SEFAZ  
✅ Retorna se está autorizada/cancelada

<img src="status.png" alt="Golang" width="700" />

---

## 🧩 Fluxo Inteligente

Fluxo lógico atual do validador:

```mermaid
graph TD
    A[Valida XSD]
    B[Erro de schema]
    C[Parse XML]
    D[XML invalido]
    E[Consulta SEFAZ]
    F[Retorna apenas dados do XML]
    G[Status real da NFe]

    A -- erro --> B
    A -- ok --> C
    C -- erro --> D
    C -- ok --> E
    E -- "skip sefaz" --> F
    E -- consulta --> G
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
