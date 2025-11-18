#!/bin/bash

# --- Configurações ---
RELEASE_URL="https://github.com/fabyo/sefaz-scraper/releases/latest/download/schemas-v4-latest.zip"
TARGET_DIR="schemas/v4"
TEMP_FILE="/tmp/schemas_v4_temp.zip"
# Diretório temporário para a extração, resolvendo o aninhamento
TEMP_EXTRACT_DIR="/tmp/schemas_extract_$RANDOM" 

# --- 1. Garante que o diretório de destino exista ---
mkdir -p "$TARGET_DIR"

# --- 2. Baixa o ZIP ---
echo "Baixando schemas da Release mais recente..."
curl -L --fail -s -o "$TEMP_FILE" "$RELEASE_URL"

if [ $? -ne 0 ]; then
    echo "❌ FALHA: Download da Release falhou. Verifique a URL ou o status da Release."
    rm -f "$TEMP_FILE" 
    exit 1
fi

# --- 3. Extrai para um local temporário ---
mkdir -p "$TEMP_EXTRACT_DIR"
echo "Extraindo para diretório temporário..."
unzip -o -qq "$TEMP_FILE" -d "$TEMP_EXTRACT_DIR"

# --- 4. CORRIGE O CAMINHO ANINHADO (O conserto!) ---
# Move o conteúdo real (que está aninhado em schemas/v4) para o diretório final.
echo "Corrigindo aninhamento e movendo arquivos para $TARGET_DIR..."
# O comando abaixo move TUDO o que está na pasta TEMP/schemas/v4 para a pasta schemas/v4
mv "$TEMP_EXTRACT_DIR"/schemas/v4/* "$TARGET_DIR"

# --- 5. Limpa ---
rm -rf "$TEMP_EXTRACT_DIR"
rm "$TEMP_FILE"

echo "✅ Sucesso! Schemas NF-e atualizados em $TARGET_DIR."
