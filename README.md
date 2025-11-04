# <img src="https://img.shields.io/badge/Go-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go"/> Golang NFE Validator 📄✅

<img src="https://img.shields.io/badge/XML%2FXSD-NFe-blueviolet?style=for-the-badge" alt="XML/XSD"/>

<img src="go-nfe.jpg" alt="Golang" width="200" />

Validador de **NF-e em Go**, usando:

- **XSD oficial da NF-e (4.00)** via `libxml2` + `go-xsd-validate`
- Suporte tanto para XML com root `<NFe>` quanto `<nfeProc>` (arquivos `*-procNFe.xml`)
- Saída em **JSON estruturado**, pronto para usar em APIs, antifraude, auditoria, etc.

A ideia é ser o **"núcleo técnico"** de um validador moderno de NF-e:

- Validação forte via XSD (schemes oficiais)
- Extração de campos relevantes
- Facilitar futura integração com:
  - SEFAZ (consulta real)
  - IA (análise de risco, explicações, etc.)

---

## 🧠 O que o projeto faz hoje

Dado um arquivo XML de NF-e ou procNFe:

- Lê o arquivo
- Valida o XML contra o **schema XSD oficial**
- Faz parse da NFe e extrai:

  - **Modelo** (`mod`)
  - **Série** (`serie`)
  - **Número** (`nNF`)
  - **CNPJ** e razão social do emitente
  - **CNPJ/CPF** e nome do destinatário
  - **Valor** total da nota (`vNF`)

- Retorna um JSON como:

```json
{
  "tipo": "nfe",
  "chave_acesso": "3519...",
  "valido_xsd": true,
  "sefaz": {
    "autorizado": true,
    "codigo": "100",
    "mensagem": "Autorizado o uso da NF-e (mock)"
  },
  "dados_xml": {
    "modelo": "55",
    "serie": "1",
    "numero": "4042",
    "emitente_cnpj": "12345678000199",
    "emitente_razao": "EMPRESA EXEMPLO LTDA",
    "destinatario_doc": "98765432000188",
    "destinatario_nome": "CLIENTE TESTE",
    "valor_total_nota": "199.90"
  }
}
