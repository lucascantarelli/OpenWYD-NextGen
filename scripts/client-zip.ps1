# Gera `client.zip` para o launcher (GitHub Releases/Drive) sem `zip` no PATH.
#
# Regras (as mesmas do manifesto):
# - entradas relativas à pasta do cliente, sem prefixo `client/`;
# - fora: manifest.txt, canais.txt, launcher.conf, WYDLauncher.exe,
#   serverlist.*, sn.bin, *.bak/*.tmp/*.part/*.pdb.
#
# Uso:
#   powershell -NoProfile -ExecutionPolicy Bypass -File scripts/client-zip.ps1
#   powershell ... -File scripts/client-zip.ps1 -Client client -Out client.zip
param(
  [string]$Client = "client",
  [string]$Out = "client.zip"
)

Add-Type -AssemblyName System.IO.Compression
Add-Type -AssemblyName System.IO.Compression.FileSystem

$raiz = (Resolve-Path $Client).Path
$destino = [System.IO.Path]::GetFullPath((Join-Path (Get-Location) $Out))
if (Test-Path $destino) { Remove-Item $destino -Force }

$excluidos = @(
  "manifest.txt", "canais.txt", "launcher.conf", "launcher.conf.bak",
  "WYDLauncher.exe", "WYDLauncher.exe.bak", "serverlist.bin", "serverlist.bin.bak",
  "serverlist.txt", "sn.bin"
)

$zip = [System.IO.Compression.ZipFile]::Open($destino, [System.IO.Compression.ZipArchiveMode]::Create)
$total = 0
try {
  Get-ChildItem -LiteralPath $raiz -Recurse -File | ForEach-Object {
    $rel = $_.FullName.Substring($raiz.Length + 1).Replace('\', '/')
    $nome = $_.Name
    $pular = ($excluidos -contains $rel) -or ($excluidos -contains $nome) -or ($nome -match '\.(bak|tmp|part|pdb)$')
    if (-not $pular) {
      [System.IO.Compression.ZipFileExtensions]::CreateEntryFromFile(
        $zip, $_.FullName, $rel, [System.IO.Compression.CompressionLevel]::Optimal
      ) | Out-Null
      $total += 1
    }
  }
} finally {
  $zip.Dispose()
}
$bytes = (Get-Item $destino).Length
Write-Output "client.zip: $total arquivo(s), $bytes byte(s) -> $destino"
