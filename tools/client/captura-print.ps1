# Captura a imagem da janela do cliente (FASE-08-6) — exige elevação, como o próprio cliente.
#
# Por que existe: `WYD.exe` desenha a própria UI em D3D9 (UIAutomation devolve 0 elementos),
# então a única forma de saber onde fica o botão do servidor é olhar a janela. Este script abre
# o cliente, espera o menu aparecer e grava dois PNGs em `target/`:
#
#   - `janela-printwindow.png`: via `PrintWindow` (conteúdo da janela, mesmo coberta);
#   - `janela-blt.png`: via `BitBlt` da tela recortada no retângulo da janela (fallback, porque
#     superfícies D3D às vezes saem pretas no PrintWindow).
#
# O cliente é encerrado no fim: a sessão de captura de bytes é outro script
# (`captura-sessao.ps1`), que clica nas coordenadas descobertas aqui.
param(
  [int]$EsperaSegundos = 30,
  [string]$Log = "target/captura-print.log"
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

Escreve "=== inicio print: raiz=$raiz espera=$EsperaSegundos ==="
Get-Process WYD -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction Continue
Start-Sleep -Seconds 2

$cli = Start-Process -FilePath (Join-Path $raiz "client/WYD.exe") -WorkingDirectory (Join-Path $raiz "client") -PassThru
Escreve "cliente: PID=$($cli.Id)"
Start-Sleep -Seconds $EsperaSegundos

Add-Type -AssemblyName System.Drawing
Add-Type -TypeDefinition @"
using System;
using System.Drawing;
using System.Runtime.InteropServices;
public class JanelaScreen {
  [DllImport("user32.dll")] public static extern bool PrintWindow(IntPtr hwnd, IntPtr hdc, uint flags);
  [DllImport("user32.dll")] public static extern bool GetWindowRect(IntPtr h, out RECT r);
  [DllImport("user32.dll")] public static extern IntPtr GetWindowDC(IntPtr h);
  [DllImport("user32.dll")] public static extern int ReleaseDC(IntPtr h, IntPtr dc);
  [DllImport("user32.dll")] public static extern bool SetForegroundWindow(IntPtr h);
  [DllImport("user32.dll")] public static extern bool ShowWindow(IntPtr h, int cmd);
  [DllImport("gdi32.dll")] public static extern bool BitBlt(IntPtr d, int x, int y, int w, int h, IntPtr s, int sx, int sy, int rop);
  [StructLayout(LayoutKind.Sequential)] public struct RECT { public int Left, Top, Right, Bottom; }
  public static RECT Retangulo(IntPtr hwnd) { RECT r; GetWindowRect(hwnd, out r); return r; }
  public static void PrintWindowPara(IntPtr hwnd, string caminho) {
    RECT r = Retangulo(hwnd);
    int w = r.Right - r.Left, h = r.Bottom - r.Top;
    using (var bmp = new Bitmap(Math.Max(w, 1), Math.Max(h, 1)))
    using (var g = Graphics.FromImage(bmp)) {
      IntPtr hdc = g.GetHdc();
      PrintWindow(hwnd, hdc, 2);
      g.ReleaseHdc(hdc);
      bmp.Save(caminho, System.Drawing.Imaging.ImageFormat.Png);
    }
  }
  public static void BltPara(IntPtr hwnd, string caminho) {
    RECT r = Retangulo(hwnd);
    int w = r.Right - r.Left, h = r.Bottom - r.Top;
    using (var bmp = new Bitmap(Math.Max(w, 1), Math.Max(h, 1)))
    using (var g = Graphics.FromImage(bmp)) {
      IntPtr hdc = g.GetHdc();
      IntPtr src = GetWindowDC(hwnd);
      BitBlt(hdc, 0, 0, w, h, src, 0, 0, 0x00CC0020);
      ReleaseDC(hwnd, src);
      g.ReleaseHdc(hdc);
      bmp.Save(caminho, System.Drawing.Imaging.ImageFormat.Png);
    }
  }
}
"@ -ReferencedAssemblies System.Drawing

$proc = Get-Process -Id $cli.Id -ErrorAction SilentlyContinue
if (-not $proc) {
  Escreve "cliente morreu antes da captura de tela"
  exit 1
}
Escreve "titulo='$($proc.MainWindowTitle)' handle=$($proc.MainWindowHandle)"
[JanelaScreen]::ShowWindow($proc.MainWindowHandle, 9) | Out-Null   # SW_RESTORE
[JanelaScreen]::SetForegroundWindow($proc.MainWindowHandle) | Out-Null
Start-Sleep -Seconds 2
$r = [JanelaScreen]::Retangulo($proc.MainWindowHandle)
Escreve "retangulo: $($r.Left),$($r.Top) $($r.Right - $r.Left)x$($r.Bottom - $r.Top)"

$pngPrint = Join-Path $raiz "target/janela-printwindow.png"
$pngBlt = Join-Path $raiz "target/janela-blt.png"
try { [JanelaScreen]::PrintWindowPara($proc.MainWindowHandle, $pngPrint); Escreve "PrintWindow -> $pngPrint" } catch { Escreve "PrintWindow falhou: $($_.Exception.Message)" }
try { [JanelaScreen]::BltPara($proc.MainWindowHandle, $pngBlt); Escreve "BitBlt -> $pngBlt" } catch { Escreve "BitBlt falhou: $($_.Exception.Message)" }

Get-Process WYD -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction Continue
Escreve "=== fim ==="
