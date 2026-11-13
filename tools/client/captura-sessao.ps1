# Sessão de captura do handshake do cliente (FASE-08-6) — exige elevação.
#
# Por que elevação: `client/WYD.exe` declara `requireAdministrator` no manifesto, então
# (a) os processos do cliente não podem ser encerrados por um shell comum ("Acesso
# negado" no taskkill) e (b) mouse/teclado de processo não elevado não atravessa a UIPI.
#
# Fluxo descoberto nas execuções de 2026-11-08 (14:07–14:16):
#   - `client/serverlist.bin` (110 registros de 64 B) aponta para 192.168.100.3/.4, que já
#     existem como endereços desta máquina; nada de rede a criar;
#   - ao abrir, o cliente faz `GET /serv00.htm` na porta 80 (capturado: 65 B,
#     `User-Agent: MS`);
#   - o TCP 8281 só abre quando a linha do servidor **e** o canal são escolhidos e o botão
#     "Conectar" é clicado (tela desenhada em D3D9, sem controles para UIAutomation).
#
# Este script: encerra clientes antigos, sobe listeners (8281 + 80), abre o cliente, tira
# screenshots de cada etapa em `target/`, seleciona servidor e canal, clica em Conectar,
# digita o login hipotético ("captura") e registra o que chegou. Não apaga endereço
# nenhum que já existia na máquina.
param(
  [int]$Porta = 8281,
  [int]$EsperaSegundos = 110,
  [string]$Artefato = "target/captura-8281.json",
  [string]$Log = "target/captura-sessao.log",
  # Coordenadas medidas em target/janela-printwindow.png (janela 1030x797 em 0,0):
  # linha "NextGen" da coluna Servidor, primeira linha da coluna Canal e botão Conectar.
  [int]$ServidorX = 397,
  [int]$ServidorY = 372,
  [int]$CanalX = 574,
  [int]$CanalY = 372,
  [int]$ConectarX = 409,
  [int]$ConectarY = 584
)

$ErrorActionPreference = "Continue"
$raiz = (Resolve-Path (Join-Path $PSScriptRoot "..\..")).Path
$logPath = Join-Path $raiz $Log
New-Item -ItemType Directory -Force -Path (Split-Path $logPath) | Out-Null
function Escreve([string]$texto) {
  $linha = "$(Get-Date -Format o) $texto"
  Write-Host $linha
  Add-Content -Path $logPath -Value $linha -Encoding utf8
}

Add-Type -AssemblyName System.Drawing
Add-Type -AssemblyName System.Windows.Forms
Add-Type -TypeDefinition @"
using System;
using System.Drawing;
using System.Runtime.InteropServices;
public class TelaCliente {
  [DllImport("user32.dll")] public static extern bool PrintWindow(IntPtr hwnd, IntPtr hdc, uint flags);
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out RECT r);
  [DllImport("user32.dll")] public static extern bool SetCursorPos(int x, int y);
  [DllImport("user32.dll")] public static extern void mouse_event(uint f, uint dx, uint dy, uint d, int e);
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr h);
  [StructLayout(LayoutKind.Sequential)] public struct RECT { public int Left, Top, Right, Bottom; }
  public static RECT Retangulo(IntPtr hwnd) { RECT r; GetWindowRect(hwnd, out r); return r; }
  public static void Clique(int x, int y) {
    SetCursorPos(x, y);
    mouse_event(0x0002, 0, 0, 0, 0);
    mouse_event(0x0004, 0, 0, 0, 0);
  }
  public static void Print(IntPtr hwnd, string caminho) {
    RECT r = Retangulo(hwnd);
    int w = Math.Max(r.Right - r.Left, 1), h = Math.Max(r.Bottom - r.Top, 1);
    using (var bmp = new Bitmap(w, h))
    using (var g = Graphics.FromImage(bmp)) {
      IntPtr hdc = g.GetHdc();
      PrintWindow(hwnd, hdc, 2);
      g.ReleaseHdc(hdc);
      bmp.Save(caminho, System.Drawing.Imaging.ImageFormat.Png);
    }
  }
}
"@ -ReferencedAssemblies System.Drawing

function Screenshot([IntPtr]$handle, [string]$nome) {
  $caminho = Join-Path $raiz "target/$nome.png"
  try { [TelaCliente]::Print($handle, $caminho); Escreve "screenshot: $caminho" } catch { Escreve "screenshot $nome falhou: $($_.Exception.Message)" }
  # versão pequena em JPEG para inspeção (PNG de 1030x797 passa do limite de leitura)
  try {
    $img = [System.Drawing.Image]::FromFile($caminho)
    $larg = 700
    $alt = [int]($img.Height * ($larg / $img.Width))
    $bmp = New-Object System.Drawing.Bitmap($larg, $alt)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.InterpolationMode = "HighQualityBicubic"
    $g.DrawImage($img, 0, 0, $larg, $alt)
    $codec = [System.Drawing.Imaging.ImageCodecInfo]::GetImageEncoders() | Where-Object { $_.MimeType -eq "image/jpeg" }
    $par = New-Object System.Drawing.Imaging.EncoderParameters(1)
    $par.Param[0] = New-Object System.Drawing.Imaging.EncoderParameter([System.Drawing.Imaging.Encoder]::Quality, 65)
    $bmp.Save((Join-Path $raiz "target/$nome.jpg"), $codec, $par)
    $g.Dispose(); $bmp.Dispose(); $img.Dispose()
  } catch { Escreve "jpeg $nome falhou: $($_.Exception.Message)" }
}

Escreve "=== inicio: raiz=$raiz porta=$Porta espera=$EsperaSegundos ==="
Escreve ("elevacao: " + ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator))

Get-Process WYD -ErrorAction SilentlyContinue | ForEach-Object {
  Escreve "encerrando WYD.exe antigo PID=$($_.Id)"
  $_ | Stop-Process -Force -ErrorAction Continue
}
# Listeners de tentativas anteriores seguram as portas 80/8281: fecha só os deste harness.
Get-CimInstance Win32_Process -Filter "Name='node.exe'" -ErrorAction SilentlyContinue |
  Where-Object { $_.CommandLine -like "*captura-live.mjs*" } |
  ForEach-Object {
    Escreve "encerrando listener antigo PID=$($_.ProcessId)"
    Stop-Process -Id $_.ProcessId -Force -ErrorAction Continue
  }
Start-Sleep -Seconds 2

foreach ($ip in @("192.168.100.3", "192.168.100.4")) {
  $saida = (& netsh interface ipv4 add address "Loopback Pseudo-Interface 1" $ip 255.255.255.255 2>&1 | Out-String).Trim()
  Escreve "netsh add $ip -> $saida"
}

$nodeExe = (Get-Command node).Source
$outLog = Join-Path $raiz "target/captura-live.out.log"
$errLog = Join-Path $raiz "target/captura-live.err.log"
$node = Start-Process -FilePath $nodeExe -ArgumentList @("tools/client/captura-live.mjs", "--host", "0.0.0.0", "--port", "$Porta", "--timeout", "$($EsperaSegundos * 1000)", "--out", $Artefato) -WorkingDirectory $raiz -PassThru -WindowStyle Hidden -RedirectStandardOutput $outLog -RedirectStandardError $errLog
Escreve "listener $Porta : PID=$($node.Id)"
$node80 = Start-Process -FilePath $nodeExe -ArgumentList @("tools/client/captura-live.mjs", "--host", "0.0.0.0", "--port", "80", "--timeout", "$($EsperaSegundos * 1000)", "--out", "target/captura-http80.json") -WorkingDirectory $raiz -PassThru -WindowStyle Hidden -RedirectStandardOutput (Join-Path $raiz "target/captura-live80.out.log") -RedirectStandardError (Join-Path $raiz "target/captura-live80.err.log")
Escreve "listener 80 : PID=$($node80.Id)"
Start-Sleep -Seconds 3

$cli = Start-Process -FilePath (Join-Path $raiz "client/WYD.exe") -WorkingDirectory (Join-Path $raiz "client") -PassThru
Escreve "cliente: PID=$($cli.Id)"
Start-Sleep -Seconds 30

$proc = Get-Process -Id $cli.Id -ErrorAction SilentlyContinue
if (-not $proc) {
  Escreve "cliente morreu antes da automação"
} else {
  $handle = $proc.MainWindowHandle
  Escreve "janela: '$($proc.MainWindowTitle)' handle=$handle"
  [TelaCliente]::SetForegroundWindow($handle) | Out-Null
  Start-Sleep -Seconds 1
  $r = [TelaCliente]::Retangulo($handle)
  Escreve "retangulo: $($r.Left),$($r.Top) $($r.Right - $r.Left)x$($r.Bottom - $r.Top)"
  Screenshot $handle "etapa1-menu"

  Escreve "clique no servidor ($ServidorX,$ServidorY)"
  [TelaCliente]::Clique($r.Left + $ServidorX, $r.Top + $ServidorY)
  Start-Sleep -Seconds 3
  Screenshot $handle "etapa2-servidor"

  Escreve "clique no canal ($CanalX,$CanalY)"
  [TelaCliente]::Clique($r.Left + $CanalX, $r.Top + $CanalY)
  Start-Sleep -Seconds 3
  Screenshot $handle "etapa3-canal"

  Escreve "clique em Conectar ($ConectarX,$ConectarY)"
  [TelaCliente]::Clique($r.Left + $ConectarX, $r.Top + $ConectarY)
  Start-Sleep -Seconds 12
  Screenshot $handle "etapa4-conectar"

  $estabelecidas = Get-NetTCPConnection -LocalPort $Porta -State Established -ErrorAction SilentlyContinue
  Escreve "conexoes ESTABELECIDAS na porta ${Porta}: $(($estabelecidas | Measure-Object).Count)"

  $wsh = New-Object -ComObject WScript.Shell
  $wsh.AppActivate($cli.Id) | Out-Null
  Escreve "teclas de login"
  $wsh.SendKeys("captura")
  Start-Sleep -Seconds 1
  $wsh.SendKeys("{TAB}")
  $wsh.SendKeys("captura")
  $wsh.SendKeys("{ENTER}")
  Start-Sleep -Seconds 10
  Screenshot $handle "etapa5-login"
}

$node.WaitForExit($EsperaSegundos * 1000 + 15000) | Out-Null
if (-not $node.HasExited) { $node | Stop-Process -Force -ErrorAction Continue }
if (-not $node80.HasExited) { $node80 | Stop-Process -Force -ErrorAction Continue }
Get-Process WYD -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction Continue
Escreve "cliente encerrado"

foreach ($arquivo in @($outLog, $errLog, (Join-Path $raiz "target/captura-live80.out.log"), (Join-Path $raiz "client/WYD.log"))) {
  if (Test-Path $arquivo) { Escreve ("--- $arquivo`n" + (Get-Content $arquivo -Raw)) }
}
Escreve "=== fim ==="
